import uuid

import mock
import pytest
import requests

from babymailgun import client
import tests


class TestToUrl(tests.TestBase):
    def test_to_url(self):
        cli = client.MailgunAPIClient("1.2.3.4", "1234")
        assert cli.to_url("emails") == "http://1.2.3.4:1234/emails"


class TestGetEmails(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        return client.MailgunAPIClient("1.2.3.4", "1234")

    def _mock(self, expected, status_code):
        mock_response = mock.MagicMock()
        mock_response.status_code = status_code
        def from_json():
            return expected

        mock_response.json = from_json
        def mock_get(*_args, **_kwargs):
            return mock_response

        return mock_get

    def test_get_emails(self, api_client):
        expected = [{"_id": str(uuid.uuid4()), "subject": "Subject"}]
        mock_get = self._mock(expected, 200)

        with mock.patch("requests.get", mock_get):
            resp = api_client.get_emails()

        assert resp == expected

    def test_get_emails_connection_error(self, api_client):
        with mock.patch("requests.get") as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError
            with pytest.raises(client.ConnectionRefused):
                api_client.get_emails()

    def test_get_emails_other_failure(self, api_client):
        mock_get = self._mock(None, 500)

        with mock.patch("requests.get", mock_get):
            with pytest.raises(client.GetFailure):
                api_client.get_emails()


class TestGetEmailById(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        return client.MailgunAPIClient("1.2.3.4", "1234")

    @pytest.fixture(scope="module")
    def email_id(self):
        return str(uuid.uuid4())

    def _mock(self, expected, status_code):
        mock_response = mock.MagicMock()
        mock_response.status_code = status_code
        def from_json():
            return expected

        mock_response.json = from_json
        def mock_get_by_id(*_args, **_kwargs):
            return mock_response

        return mock_get_by_id

    def test_get_email_by_id(self, email_id, api_client):
        expected = {"_id": email_id, "subject": "Subject"}
        mock_get = self._mock(expected, 200)

        with mock.patch("requests.get", mock_get):
            resp = api_client.get_email_by_id(email_id)

        assert resp == expected

    def test_get_email_by_id_connection_error(self, email_id, api_client):
        with mock.patch("requests.get") as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError
            with pytest.raises(client.ConnectionRefused):
                api_client.get_email_by_id(email_id)

    def test_get_email_by_id_not_found(self, email_id, api_client):
        mock_get = self._mock(None, 404)

        with mock.patch("requests.get", mock_get):
            with pytest.raises(client.NotFound):
                api_client.get_email_by_id(email_id)

    def test_get_email_by_id_other_failure(self, email_id, api_client):
        mock_get = self._mock(None, 500)

        with mock.patch("requests.get", mock_get):
            with pytest.raises(client.GetFailure):
                api_client.get_email_by_id(email_id)


class TestGetEmailRecipients(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        return client.MailgunAPIClient("1.2.3.4", "1234")

    @pytest.fixture(scope="module")
    def email_id(self):
        return str(uuid.uuid4())

    def _mock(self, expected, status_code):
        mock_response = mock.MagicMock()
        mock_response.status_code = status_code
        def from_json():
            return expected

        mock_response.json = from_json
        def mock_get_recipients(*_args, **_kwargs):
            return mock_response

        return mock_get_recipients

    def test_get_email_by_id(self, email_id, api_client):
        expected = {"recipients": [{"to": "to@unittests.com"}]}
        mock_get = self._mock(expected, 200)

        with mock.patch("requests.get", mock_get):
            resp = api_client.get_email_recipients(email_id)

        assert resp == expected

    def test_get_email_by_id_connection_error(self, email_id, api_client):
        with mock.patch("requests.get") as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError
            with pytest.raises(client.ConnectionRefused):
                api_client.get_email_recipients(email_id)

    def test_get_email_by_id_not_found(self, email_id, api_client):
        mock_get = self._mock(None, 404)

        with mock.patch("requests.get", mock_get):
            with pytest.raises(client.NotFound):
                api_client.get_email_recipients(email_id)

    def test_get_email_by_id_other_failure(self, email_id, api_client):
        mock_get = self._mock(None, 500)

        with mock.patch("requests.get", mock_get):
            with pytest.raises(client.GetFailure):
                api_client.get_email_recipients(email_id)


class TestCreateEmail(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        return client.MailgunAPIClient("1.2.3.4", "1234")

    def create_signature(self):
        subject = "Email Subject"
        sender = "sender@unittests.com"
        to = ["to@unittests.com"]
        cc = ["cc@unittests.com"]
        bcc = ["bcc@unittests.com"]
        email_body = "Buffalo" * 8
        return {"subject": subject, "sender": sender, "to": to,
                "cc": cc, "bcc": bcc, "email_body": email_body}

    def _mock(self, expected, status_code):
        mock_response = mock.MagicMock()
        mock_response.status_code = status_code
        def from_json():
            return expected

        mock_response.json = from_json
        def mock_create_email(*_args, **_kwargs):
            return mock_response

        return mock_create_email

    def test_create_email(self, api_client):
        signature = self.create_signature()

        expected = {"subject": signature["subject"],
                    "sender": signature["sender"],
                    "to": signature["to"],
                    "cc": signature["cc"],
                    "bcc": signature["bcc"],
                    "email_body": signature["email_body"]}

        mock_post = self._mock(expected, 200)

        with mock.patch("requests.post", mock_post):
            resp = api_client.create_email(**signature)

        assert resp == expected

    def test_create_email_connection_error(self, api_client):
        with mock.patch("requests.post") as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError
            with pytest.raises(client.ConnectionRefused):
                api_client.create_email(**self.create_signature())

    def test_create_email_other_failure(self, api_client):
        mock_get = self._mock(None, 500)

        with mock.patch("requests.post", mock_get):
            with pytest.raises(client.CreateFailure):
                api_client.create_email(**self.create_signature())


class TestDeleteEmail(tests.TestBase):
    @pytest.fixture()
    def api_client(self):
        return client.MailgunAPIClient("1.2.3.4", "1234")

    @pytest.fixture(scope="module")
    def email_id(self):
        return str(uuid.uuid4())

    def _mock(self, status_code):
        mock_response = mock.MagicMock()
        mock_response.status_code = status_code
        def mock_delete(*_args, **_kwargs):
            return mock_response

        return mock_delete

    def test_delete_email(self, email_id, api_client):
        mock_get = self._mock(204)

        with self.not_raises():
            with mock.patch("requests.delete", mock_get):
                api_client.delete_email(email_id)

    def test_get_email_by_id_connection_error(self, email_id, api_client):
        with mock.patch("requests.delete") as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError
            with pytest.raises(client.ConnectionRefused):
                api_client.delete_email(email_id)

    def test_get_email_by_id_not_found(self, email_id, api_client):
        mock_get = self._mock(404)

        with mock.patch("requests.delete", mock_get):
            with pytest.raises(client.NotFound):
                api_client.delete_email(email_id)

    def test_get_email_by_id_other_failure(self, email_id, api_client):
        mock_get = self._mock(500)

        with mock.patch("requests.delete", mock_get):
            with pytest.raises(client.DeleteFailure):
                api_client.delete_email(email_id)
