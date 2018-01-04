import datetime
from dateutil import parser
import os

import pytest

from babymailgun import client
import tests


class TestAPI(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        host = os.environ.get("API_HOST", "127.0.0.1")
        port = os.environ.get("API_PORT", "5000")
        return client.MailgunAPIClient(host, port)

    def test_queue_and_fetch_email(self, api_client):
        subject = "Test Subject"
        sender = "me@user.io"
        to = ["to@functional.biz"]
        cc = ["cc@functional.biz"]
        bcc = ["bcc@functional.biz"]
        email_body = "Coming to you live from Functional Suite 2018!"
        email = api_client.create_email(subject, sender, to, cc,
                                        bcc, email_body)
        assert "id" in email
        assert email["sender"] == sender
        assert email["body"] == email_body
        assert email["status"] in ("incomplete", "complete", "failed")
        assert "reason" in email
        assert "tries" in email and email["tries"] >= 0
        assert isinstance(parser.parse(email["created_at"]), datetime.datetime)
        assert isinstance(parser.parse(email["updated_at"]), datetime.datetime)

        try:
            shown = api_client.get_email_by_id(email["id"])
            assert shown["id"] == email["id"]
            assert shown["sender"] == sender
            assert shown["body"] == email_body
            assert shown["status"] in ("incomplete", "complete", "failed")
            assert "reason" in shown
            assert "tries" in shown and shown["tries"] >= 0

            recipients = api_client.get_email_recipients(email["id"])
            expected_recipients = {
                to[0]: "to",
                cc[0]: "cc",
                bcc[0]: "bcc"}

            assert len(recipients) == len(expected_recipients)
            for recipient in recipients:
                assert recipient["address"] in expected_recipients
                assert recipient["type"] == \
                    expected_recipients[recipient["address"]]
                assert recipient["reason"] == ""

            fetched = api_client.get_emails()
            for f in fetched:
                if f["id"] == email["id"]:
                    assert f["sender"] == sender
                    assert f["status"] in ("incomplete", "complete", "failed")
                    assert "tries" in f and f["tries"] >= 0
                    assert f["created_at"] == email["created_at"]
        finally:
            api_client.delete_email(email["id"])

    def test_show_email_invalid_id_404s(self, api_client):
        try:
            api_client.get_email_by_id("foo")
        except client.NotFound:
            pass
        except Exception as e:
            pytest.fail("Expected NotFound, instead got {}".format(e))

    def test_get_recipients_invalid_id_404s(self, api_client):
        try:
            api_client.get_email_recipients("foo")
        except client.NotFound:
            pass
        except Exception as e:
            pytest.fail("Expected NotFound, instead got {}".format(e))

    def test_delete_invalid_id_404s(self, api_client):
        try:
            api_client.delete_email("foo")
        except client.NotFound:
            pass
        except Exception as e:
            pytest.fail("Expected NotFound, instead got {}".format(e))
