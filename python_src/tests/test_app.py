import os

import pytest

from babymailgun import app as mailgun_app
import tests


class TestApp(tests.TestBase):
    @pytest.fixture()
    def app(self):
        mailgun_app.app.testing = True
        mailgun_app.app.config["PRESERVE_CONTEXT_ON_EXCEPTION"] = True
        os.environ["DB_HOST"] = os.environ.get("DB_HOST", "127.0.0.1")
        os.environ["DB_NAME"] = os.environ.get("DB_NAME", "mailgun")
        os.environ["DB_PORT"] = os.environ.get("DB_PORT", "27017")

        yield mailgun_app.app.test_client()
        mailgun_app.app.testing = False

    def test_list_emails(self, app):
        print(app.get("/emails"))
