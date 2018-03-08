import os
import uuid

import pymongo


def get_env(key):
    if key not in os.environ:
        raise ConfigKeyNotFound(key=key)
    return os.environ[key]


def _get_db_client():
    client = pymongo.MongoClient(DB_HOST, DB_PORT)
    db = client[DB_NAME]
    return client, db


DB_HOST = get_env("DB_HOST")
try:
    DB_PORT = int(get_env("DB_PORT"))
except ValueError:
    raise ConfigTypeError(key="DB_PORT", key_type="int")

DB_NAME = get_env("DB_NAME")


def add_smtp_server_if_not_exist():
    # More suitable as an entrypoint but we don't need one for anything else
    # so it's best here for simplicity's sake
    client, db = _get_db_client()
    server = db.servers.find_one({})
    if not server:
        server_id = str(uuid.uuid4())
        data = {
                "_id": server_id,
                "hostname": "127.0.0.1",
                "port": 1025,
                "username": "admin@mailgun.com",
                "password": "password"
                }
        db.servers.insert_one(data)

add_smtp_server_if_not_exist()
