import json

import requests


class ClientError(Exception):
    def  __init__(self, **keys):
        super().__init__(self.message % keys)


class ConnectionRefused(ClientError):
    message = "Server %(host)s:%(port)s refused the connection"


class NotFound(ClientError):
    message = "The requested resource %(resource)s was not found"


class GetFailure(ClientError):
    message = "Fetching %(resource)s failed with HTTP %(code)s: %(reason)s"


class CreateFailure(ClientError):
    message = "Creating %(resource)s failed with HTTP %(code)s: %(reason)s"


class MailgunAPIClient(object):
    def __init__(self, host, port):
        self._host = host
        self._port = port

    def to_url(self, resource):
        return "http://{}:{}/{}".format(self._host, self._port, resource)

    def get_emails(self):
        headers = {"Accept": "application/json"}
        try:
            resp = requests.get(self.to_url("emails"), headers=headers)
        except requests.exceptions.ConnectionError:
            raise ConnectionRefused(host=self._host, port=self._port)

        if resp.status_code != 200:
            raise GetFailure(resource="/emails", code=resp.status_code,
                             reason=resp.text)

        return resp.json()

    def get_email_by_id(self, email_id):
        headers = {"Accept": "application/json"}
        try:
            resp = requests.get(self.to_url("emails/{}".format(email_id)),
                                headers=headers)
        except requests.exceptions.ConnectionError:
            raise ConnectionRefused(host=self._host, port=self._port)

        if resp.status_code == 404:
            raise NotFound(resource="/emails/{}".format(email_id))

        if resp.status_code != 200:
            raise GetFailure(resource="/emails/{}".format(email_id),
                             code=resp.status_code,
                             reason=resp.text)

        return resp.json()

    def get_email_recipients(self, email_id):
        headers = {"Accept": "application/json"}
        try:
            resp = requests.get(
                self.to_url("emails/{}/recipients".format(email_id)),
                headers=headers)
        except requests.exceptions.ConnectionError:
            raise ConnectionRefused(host=self._host, port=self._port)

        if resp.status_code == 404:
            raise NotFound(resource="/emails/{}/recipients".format(email_id))

        if resp.status_code != 200:
            raise GetFailure(resource="/emails/{}/recipients".format(email_id),
                             code=resp.status_code,
                             reason=resp.text)

        return resp.json()

    def create_email(self, subject, sender, to, cc, bcc, email_body):
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}

        data = json.dumps({"subject": subject, "from": sender,
                           "to": to, "cc": cc, "bcc": bcc, "body": email_body})

        try:
            resp = requests.post(self.to_url("emails"), headers=headers,
                                 data=data)
        except requests.exceptions.ConnectionError:
            raise ConnectionRefused(host=self._host, port=self._port)

        if resp.status_code != 200:
            raise CreateFailure(resource="/emails",
                                code=resp.status_code,
                                reason=resp.text)
        return resp.json()
