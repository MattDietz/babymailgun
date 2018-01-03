import json
import os
import sys

import click
import prettytable
import requests


@click.group()
def email_cli():
    pass

@email_cli.command(help="Fetch emails")
def get():
    headers = {"Accept": "application/json"}
    try:
        resp = requests.get("http://127.0.0.1:5000/emails", headers=headers)
    except requests.exceptions.ConnectionError:
        click.echo("Failed to fetch emails. Server refused tbe connection")
        return

    if not resp.status_code == 200:
        click.echo("GET /emails failed")
        click.echo("Status code:", resp.status_code)
        click.echo(resp.text)
        return

    table = prettytable.PrettyTable()
    table.field_names = ["Id", "Sender", "Status", "Reason",
                         "Created", "Updated", "Sending Attempts"]
    emails = resp.json()
    for email in emails:
        table.add_row([email["id"], email["sender"], email["status"],
                       email["reason"], email["created_at"],
                       email["updated_at"], email["tries"]])
    click.echo(str(table))


@email_cli.command(help="Get details of a single email")
@click.argument("email_id")
@click.option("-s", "--show-body", is_flag=True, default=False,
              help="Display the entire body, regardless of length")
def show(email_id, show_body):
    headers = {"Accept": "application/json"}
    try:
        resp = requests.get("http://127.0.0.1:5000/emails/{}".format(email_id),
                            headers=headers)
    except requests.exceptions.ConnectionError:
        click.echo("Failed to fetch emails. Server refused tbe connection")
        return

    if not resp.status_code == 200:
        click.echo("GET /emails/{} failed".format(email_id))
        click.echo("Status code:", resp.status_code)
        click.echo(resp.text)
        return

    email = resp.json()
    table = prettytable.PrettyTable()
    table.field_names = ["Field", "Entry"]
    table.add_row(["ID", email["id"]])
    table.add_row(["Sender", email["sender"]])
    # We deliberately truncate the body here
    max_length = 120
    if not show_body and len(email["body"]) > max_length:
        email["body"] = email["body"][:max_length]

    table.add_row(["Body", email["body"]])
    table.add_row(["Status", email["status"]])
    table.add_row(["Reason", email["reason"]])
    table.add_row(["Created", email["created_at"]])
    table.add_row(["Updated", email["updated_at"]])
    table.add_row(["Tries", email["tries"]])
    click.echo(str(table))


@email_cli.command(help="Show recipient status of a single email")
@click.argument("email_id")
def get_recipients(email_id):
    headers = {"Accept": "application/json"}
    try:
        resp = requests.get(
            "http://127.0.0.1:5000/emails/{}/recipients".format(email_id),
            headers=headers)
    except requests.exceptions.ConnectionError:
        click.echo("Failed to fetch emails. Server refused tbe connection")
        return

    if not resp.status_code == 200:
        click.echo("GET /emails/{}/recipients failed".format(email_id))
        click.echo("Status code:", resp.status_code)
        click.echo(resp.text)
        return

    recipients = resp.json()
    table = prettytable.PrettyTable()
    table.field_names = ["Recipient", "Type", "Reason"]
    for recipient in recipients:
        table.add_row([recipient["address"], recipient["type"],
                       recipient["reason"]])
    click.echo(str(table))


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
    try:
        resp = requests.post("http://127.0.0.1:5000/emails", headers=headers,
                             data=data)
    except requests.exceptions.ConnectionError:
        click.echo("Failed to fetch emails. Server refused tbe connection")
        return

    if not resp.status_code == 200:
        click.echo("POST /emails failed")
        click.echo("Status code:", resp.status_code)
        click.echo(resp.text)
    email = resp.json()
    click.echo("Email from '{}' to '{}' with id {} queued "
               "for delivery".format(email["sender"],
                                     ", ".join([e["address"] for e in email["recipients"]]),
                                     email["id"]))


def main():
    email_cli()


if __name__ == "__main__":
    main()
