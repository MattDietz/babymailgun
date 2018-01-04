import os
import sys

import click
import prettytable

from babymailgun import client


@click.group()
def email_cli():
    pass


def get_client():
    host = os.environ.get("API_HOST", "127.0.0.1")
    port = os.environ.get("API_PORT", "5000")
    return client.MailgunAPIClient(host, port)


@email_cli.command(help="Fetch emails")
def get():
    try:
        api_client = get_client()
        emails = api_client.get_emails()
    except Exception as e:
        click.echo("Fetching emails failed with:")
        sys.exit(e)

    table = prettytable.PrettyTable()
    table.field_names = ["Id", "Sender", "Status", "Reason",
                         "Created", "Updated", "Sending Attempts"]
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
    try:
        api_client = get_client()
        email = api_client.get_email_by_id(email_id)
    except Exception as e:
        click.echo("Fetching emails failed with:")
        sys.exit(e)

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


@email_cli.command(help="Delete an email. This won't magically unsend "
                        "an email ;-)")
@click.argument("email_id")
def delete(email_id):
    try:
        api_client = get_client()
        api_client.delete_email(email_id)
    except Exception as e:
        click.echo("Deleting email {} failed with:".format(email_id))
        sys.exit(e)
    click.echo("Successfully deleted email with ID '{}'".format(email_id))


@email_cli.command(help="Show recipient status of a single email")
@click.argument("email_id")
def get_recipients(email_id):
    try:
        api_client = get_client()
        recipients = api_client.get_email_recipients(email_id)
    except Exception as e:
        click.echo("Fetching recipients failed with:")
        sys.exit(e)

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

    try:
        api_client = get_client()
        email = api_client.create_email(subject, sender, to, cc, bcc, email_body)
    except Exception as e:
        click.echo("Creating an email failed with:")
        sys.exit(e)

    click.echo("Email from '{}' to '{}' with id {} queued "
               "for delivery".format(
                   email["sender"],
                   ", ".join([e["address"] for e in email["recipients"]]),
                   email["id"]))


def main():
    email_cli()


if __name__ == "__main__":
    main()
