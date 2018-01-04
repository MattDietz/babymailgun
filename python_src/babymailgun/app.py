import datetime
import os
import uuid

import flask
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


app = flask.Flask(__name__)


@app.before_first_request
def setup_app():
    app.config["DB_HOST"] = get_env("DB_HOST")
    try:
        app.config["DB_PORT"] = int(get_env("DB_PORT"))
    except ValueError:
        raise ConfigTypeError(key="DB_PORT", key_type="int")

    app.config["DB_NAME"] = get_env("DB_NAME")


def to_email_model(email_id, email_dict):
    recipients = []

    def to_recipients(recipients, recipient_type):
        return [{"address": a,
                 "type": recipient_type,
                 "status": 0,
                 "reason": ""} for a in recipients]

    for receiver_type in ["to", "cc", "bcc"]:
        recipients.extend(to_recipients(email_dict[receiver_type],
                                        receiver_type))

    return {"_id": email_id,
            "headers": [],
            "subject": email_dict["subject"],
            "body": email_dict["body"],
            "sender": email_dict["from"],
            "recipients": recipients,
            "created_at": datetime.datetime.now(),
            "updated_at": datetime.datetime.fromtimestamp(0),
            "status": "incomplete",
            "reason": "",
            "tries": 0,
            "worker_id": None}


def _get_db_client():
    client = pymongo.MongoClient(app.config["DB_HOST"], app.config["DB_PORT"])
    db = client[app.config["DB_NAME"]]
    return db


@app.route("/emails", methods=["GET"])
def list_emails():
    app.logger.debug("GET /emails")
    db = _get_db_client()
    emails = []
    for email in db.emails.find():
        emails.append({
            "id": email["_id"],
            "sender": email["sender"],
            "status": email["status"],
            "reason": email["reason"],
            "created_at": email["created_at"],
            "updated_at": email["updated_at"],
            "tries": email["tries"]})
    return flask.jsonify(emails)


@app.route("/emails/<email_id>", methods=["GET"])
def show_email(email_id):
    app.logger.debug("GET /emails/%s", email_id)
    db = _get_db_client()
    email = db.emails.find_one({"_id": email_id})
    if not email:
        return ("", 404)

    email = {"id": email["_id"],
             "sender": email["sender"],
             "status": email["status"],
             "reason": email["reason"],
             "body": email["body"],
             "created_at": email["created_at"],
             "updated_at": email["updated_at"],
             "tries": email["tries"]}
    return flask.jsonify(email)


@app.route("/emails/<email_id>/recipients", methods=["GET"])
def show_email_recipients(email_id):
    # NOTE(mdietz): In the real world we'd stick a cache+rate limiting of some
    #               kind here as users would hammer this endpoint
    app.logger.debug("GET /emails/%s/recipients", email_id)
    db = _get_db_client()
    email = db.emails.find_one({"_id": email_id})
    if not email:
        return ("", 404)

    recipients = []
    for recipient in email["recipients"]:
        recipients.append({"address": recipient["address"],
                           "type": recipient["type"],
                           "reason": recipient["reason"]})
    return flask.jsonify(recipients)


@app.route("/emails", methods=["POST"])
def send_email():
    app.logger.debug("POST /emails")
    headers = flask.request.headers
    if ("content-type" not in headers or
            ("content-type" in headers and
             headers["content-type"].lower() != "application/json")):
        return ("Invalid content-type or no content-type specified", 415)

    data = flask.request.get_json()

    db = _get_db_client()
    email_id = str(uuid.uuid4())
    email = to_email_model(email_id, data)
    db.emails.insert_one(email)
    email.pop("_id")
    email["id"] = email_id

    return flask.jsonify(email)


@app.route("/emails/<email_id>", methods=["DELETE"])
def delete_email(email_id):
    app.logger.debug("DELETE /emails/%s", email_id)
    db = _get_db_client()
    delete_result = db.emails.delete_one({"_id": email_id})
    if delete_result.deleted_count == 0:
        return ("", 404)

    return ("", 204)
