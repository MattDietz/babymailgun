import datetime
import os
import uuid

import flask
import prettytable
import pymongo


class MailgunException(Exception):
    def  __init__(self, **keys):
        super().__init__(self.message % keys)


class ConfigKeyNotFound(MailgunException):
    message = ("The requested key '%(key)s' does not exist in the "
               "application configuration")


class ConfigTypeError(MailgunException):
    message = ("The key '%(key)s' must be of type %(key_type)s")


def get_env(key):
    if key not in os.environ:
        raise ConfigKeyNotFound(key=key)
    return os.environ[key]


DB_HOST = get_env("DB_HOST")
try:
    DB_PORT = int(get_env("DB_PORT"))
except ValueError:
    raise ConfigTypeError(key="DB_PORT", key_type="int")

DB_NAME = get_env("DB_NAME")
app = flask.Flask("mailgun")


def to_email_model(email_id, email_dict):
    recipients = []

    def to_recipients(recipients, recipient_type):
        return [{"address": a,
                 "type": recipient_type,
                 "status_code": 0,
                 "status_reason": ""} for a in recipients]

    for receiver_type in ["to", "cc", "bcc"]:
        recipients.extend(to_recipients(email_dict[receiver_type],
                                        receiver_type))

    return {
            "_id": email_id,
            "headers": [],
            "subject": email_dict["subject"],
            "body": email_dict["body"],
            "sender": email_dict["from"],
            "recipients": recipients,
            "created_at": datetime.datetime.now(),
            "updated_at": datetime.datetime.fromtimestamp(0),
            "status": "incomplete",
            "status_reason": "",
            "tries": 0,
            "worker_id": None
            }


@app.route("/")
def index():
    return ""


def _get_db_client():
    client = pymongo.MongoClient(DB_HOST, DB_PORT)
    db = client[DB_NAME]
    return client, db


@app.route("/emails", methods=["GET"])
def list_emails():
    app.logger.debug("GET /emails")
    client, db = _get_db_client()
    table = prettytable.PrettyTable()
    table.field_names = ["Id", "Sender", "Status", "Reason",
                         "Created", "Updated", "Sending Attempts"]
    for email in db.emails.find():
        app.logger.info(email)
        table.add_row([email["_id"], email["sender"], email["status"],
                       email["status_reason"], email["created_at"],
                       email["updated_at"], email["tries"]])
    return "{}\n".format(str(table))


@app.route("/emails/<id>", methods=["GET"])
def show_email(id):
    app.logger.debug("GET /emails/{}".format(id))
    client, db = _get_db_client()
    table = prettytable.PrettyTable()
    table.field_names = ["Field", "Entry"]
    email = db.emails.find_one({"_id": id})
    if not email:
        return ("", 404)

    table.add_row(["Id", email["_id"]])
    table.add_row(["Sender", email["sender"]])
    table.add_row(["Body", email["body"]])
    table.add_row(["Status", email["status"]])
    table.add_row(["Reason", email["status_reason"]])
    table.add_row(["Created", email["created_at"]])
    table.add_row(["Updated", email["updated_at"]])
    table.add_row(["Tries", email["tries"]])
    return "{}\n".format(str(table))


@app.route("/emails/<id>/status", methods=["GET"])
def show_email_status(id):
    # NOTE(mdietz): In the real world we'd stick a cache+rate limiting of some
    #               kind here as users would hammer this endpoint
    app.logger.debug("GET /emails/{}/status".format(id))
    client, db = _get_db_client()
    table = prettytable.PrettyTable()
    table.field_names = ["Recipient", "Type", "Status", "Reason"]
    email = db.emails.find_one({"_id": id})
    if not email:
        return ("", 404)

    for recipient in email["recipients"]:
        table.add_row([recipient["address"], recipient["type"],
                       recipient["status_code"], recipient["status_reason"]])
    return "{}\n".format(str(table))


@app.route("/emails", methods=["POST"])
def send_email():
    app.logger.debug("POST /emails")
    # TODO This should generate an MD5/SHA and store that in the db, compare to
    #      others and return a 409 or 422 , "You've already queued this email"
    headers = flask.request.headers
    if "content-type" not in headers or ("content-type" in headers and
            headers["content-type"].lower() != "application/json"):
        return ("Invalid content-type or no content-type specified", 415)

    data = flask.request.get_json()

    app.logger.debug("Sender: {}".format(data.get("from")))
    app.logger.debug("Recipients: {}".format(data.get("to")))
    app.logger.debug("Subject: {}".format(data.get("subject")))

    client, db = _get_db_client()
    email_id = str(uuid.uuid4())
    db.emails.insert_one(to_email_model(email_id, data))

    return ("Email from '{}' to '{}' with id {} queued "
            "for delivery".format(data["from"], data["to"], email_id))
