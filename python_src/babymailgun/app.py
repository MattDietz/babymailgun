import flask

app = flask.Flask("mailgun")


@app.route("/")
def index():
    return ""


@app.route("/emails", methods=["GET"])
def get_emails():
    return ""


@app.route("/emails", methods=["POST"])
def send_email(data):
    return ""

