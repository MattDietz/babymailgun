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


DB_HOST = get_env("DB_HOST")
try:
    DB_PORT = int(get_env("DB_PORT"))
except ValueError:
    raise ConfigTypeError(key="DB_PORT", key_type="int")

DB_NAME = get_env("DB_NAME")
app = flask.Flask("mailgun")


def to_email_model(email_id, email_dict):
    return {
            "_id": email_id,
            "headers": [],
            "subject": email_dict["subject"],
            "body": email_dict["body"],
            "sender": email_dict["from"],
            "recipients": [{"address": a,
                            "status_code": 0,
                            "status_reason": ""} for a in email_dict["to"]],
            "created_at": datetime.datetime.now(),
            "updated_at": datetime.datetime.now(),
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
def get_emails():
    app.logger.debug("GET /emails")
    client, db = _get_db_client()
    return str(list(db.emails.find()))


@app.route("/emails", methods=["POST"])
def send_email():
    app.logger.debug("POST /emails")
    headers = flask.request.headers
    app.logger.info(headers)
    if "content-type" not in headers or ("content-type" in headers and
            headers["content-type"].lower() != "application/json"):
        return ("Invalid content-type or no content-type specified", 415)

    data = flask.request.get_json()

    app.logger.debug("Sender: {}".format(data.get("from")))
    app.logger.debug("Recipients: {}".format(data.get("to")))
    app.logger.debug("Subject: {}".format(data.get("subject")))

    client, db = _get_db_client()
    email_id = str(uuid.uuid4())
    db.emails.insert_one(to_email_model(str(uuid.uuid4()), data))
    return "Email from '{}' queued for delivery".format(data["from"])
