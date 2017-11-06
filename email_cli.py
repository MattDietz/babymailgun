import json
import os
import sys

import click
import requests


@click.group()
def email_cli():
    pass


@email_cli.command(help="Send an email")
@click.argument("sender")
@click.option("-t", "--to", multiple=True)
@click.option("-c", "--cc", multiple=True)
@click.option("--bcc", multiple=True)
@click.option("-s", "--subject")
@click.option("-b", "--body", help="Path to a file containing the body")
def send(sender, to, cc, bcc, subject, body):
    if not body:
        sys.exit("Body path must be supplied with -b/--body!")
    body = os.path.expanduser(body)
    if body and not os.path.exists(body):
        sys.exit("Body path '{}' does not exist".format(body))
    if not to:
        sys.exit("No recipients found!")
    if not subject:
        sys.exit("No subject specified!")

    with open(body, 'r') as f:
        email_body = "".join(f.readlines())

    to = list(to)
    data = json.dumps({"subject": subject, "from": sender,
                       "to": to, "cc": cc, "bcc": bcc, "body": email_body})
    headers = {"Content-Type": "application/json",
               "Accept": "application/json"}
    resp = requests.post("http://127.0.0.1:5000/emails", headers=headers,
                         data=data)

    if not resp.status_code == 200:
        print("POST /emails failed")
        print("Status code:", resp.status_code)
    print(resp.text)


def main():
    email_cli()


if __name__ == "__main__":
    main()
