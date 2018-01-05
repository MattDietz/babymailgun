import datetime
import os
import re
import uuid

import flask
import pymongo

MAX_RECIPIENTS = 100
MAX_SUBJECT_LENGTH = 255
MAX_BODY_LENGTH = 16384

# From http://emailregex.com/
EMAIL_REGEX = re.compile(r"(^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$)")
SUBJECT_REGEX = re.compile(r"^[a-zA-Z0-9 ]*$")


class MailgunException(Exception):
    def  __init__(self, **keys):
        super().__init__(self.message % keys)


class ConfigKeyNotFound(MailgunException):
    message = ("The requested key '%(key)s' does not exist in the "
               "application configuration")


class ConfigTypeError(MailgunException):
    message = "The key '%(key)s' must be of type %(key_type)s"


class TooManyRecipients(MailgunException):
    message = ("The number of recipients for any given email may not "
               "exceed {}".format(MAX_RECIPIENTS))


class SubjectTooLong(MailgunException):
    message = ("The length of the subject may not exceed {} "
               "characters".format(MAX_SUBJECT_LENGTH))


class BodyTooLong(MailgunException):
    message = ("The length of the body may not exceed {} "
               "characters".format(MAX_BODY_LENGTH))


class InvalidEmailAddress(MailgunException):
    message = "The email '%(email)s' in the %(header)s header is invalid"


class InvalidSubject(MailgunException):
    message = ("The subject contains invalid characters. Only a-z, "
               "A-Z and 0-9 are allowed")


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


def validate_email(email_dict):
    # these are not limits imposed by any RFC, but rather are
    # here simply to keep things sane
    to = email_dict["to"]
    cc = email_dict["cc"]
    bcc = email_dict["bcc"]
    if len(to) + len(cc) + len(bcc) > MAX_RECIPIENTS:
        raise TooManyRecipients()

    if len(email_dict["subject"]) > MAX_SUBJECT_LENGTH:
        raise SubjectTooLong()

    if len(email_dict["body"]) > MAX_BODY_LENGTH:
        raise BodyTooLong()

    # I realize this is arbitrarily limiting, but for ease
    # of printing on a command line I decided it was necessary/useful
    if not SUBJECT_REGEX.match(email_dict["subject"]):
        raise InvalidSubject()

    if not EMAIL_REGEX.match(email_dict["from"]):
        raise InvalidEmailAddress(email=email_dict["from"],
                                  header="from")

    for header in ["to", "cc", "bcc"]:
        for email in email_dict[header]:
            if not EMAIL_REGEX.match(email):
                raise InvalidEmailAddress(email=email, header=header)


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

    try:
        validate_email(data)
    except Exception as e:
        return (str(e), 400)

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
