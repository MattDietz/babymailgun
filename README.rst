===========
Babymailgun
===========

This is my dev test submission for Mailgun.

The project is implemented using a combination of REST API and asynchronous worker architecture all backed by MongoDB.
The API allows both sending of emails as well fetching the status of emails that have been created.
Additionally provided is a CLI tool to make interaction with the API easier.

==========
Jumping In
==========

You'll need `Docker CE <https://docs.docker.com/engine/installation/>`_, `Docker Compose <https://docs.docker.com/compose/install/>`_ and make

A Makefile is included in the project root to help streamline playing with the project, and has it's own help text to faciliate navigation.

.. code-block:: bash

    make help

Clone the project:

.. code-block:: bash

    git clone https://github.com/Cerberus98/babymailgun.git

To kick off the compose environment, simply run the following from the root of your clone

.. code-block:: bash

    make run && make logs

Docker-compose will build two containers and fetch two additional ones which may take a while depending on your connection. Once it's done, you
should have a running environment.

To stop all of the running containers, CTRL+C out of the running log output and then:

.. code-block:: bash

    make stop

===============
Running the CLI
===============

There are two ways to setup and use the CLI

The Easy Way
============

I've created a separate Makefile task just for running the CLI against Python 3.6 which makes it pretty easy to jump in and play around

From the project root, first make sure the Docker Compose environment is running as above. Then:

.. code-block:: bash

    make shell

The Hard Way
============

If you have pyenv installed, install and switch to Python 3.6.0 or better.

For example:

.. code-block:: bash

    pyenv install 3.6.0

Then:

.. code-block:: bash

    pyenv local 3.6.0

Once Python 3.6+ is installed, you'll need to install the project and dependencies. I highly recommend a virtualenv, and recommend Virtualenvwrapper_ doubly so.
 
.. _Virtualenvwrapper: https://virtualenvwrapper.readthedocs.io/en/latest/

The instructions that follow use virtualenvwrapper.

.. code-block:: bash

    ~> mkvirtualenv mailgun
    # stuff happens
    ~> cd <project root>
    ~> make install_python

This will pip install all of the requirements, the project itself as editable, and then kick off the tests.

The Mailgun CLI
===============

The CLI binary exposes the following commands:

.. code-block:: bash

    ~> mailgun_cli

    Usage: mailgun_cli [OPTIONS] COMMAND [ARGS]...

    Options:
      --help  Show this message and exit.

    Commands:
      get             Fetch emails
      get_recipients  Show recipient status of a single email
      send            Send an email
      show            Get details of a single email

Help for specific commands is available via the --help switch. For example:

.. code-block:: bash

    ~> mailgun_cli send --help
    Usage: mailgun_cli send [OPTIONS] SENDER

      Send an email

    Options:
      -t, --to TEXT
      -c, --cc TEXT
      --bcc TEXT
      -s, --subject TEXT
      -b, --body TEXT     Path to a file containing the body
      --help              Show this message and exit.

==============
Sending Emails
==============

Create a file to represent the body of your email:

.. code-block:: bash

    touch body.txt && <editor> body.txt

Next, issue the following command, which will send an email from bob<at>mailgun.com to a<at>mailgun.com, CC emily<at>mailgun, BCC frank<at>mailgun
and will have the subject "Dinner plans":

.. code-block:: bash

    mailgun_cli send bob@mailgun.com --body ../body.txt -t matt@mailgun.com -c emily@mailgun.com --bcc frank@mailgun.com -s "Dinner plans"

If your email sends successfully through (watch the docker logs and try out the CLI) then you can open up your local Mailhog instance in your browser to see it. Simply go to http://127.0.0.1:8025 in your browser to check it out.

==============================
Setting up to run Python Tests
==============================

If you're comfortable using an interactive shell via the container as per `The Easy Way`_ above, then simply run the following in the shell:

.. code-block:: bash

    tox

Alternatively, If you've already followed `The Hard Way`_ above, you've already got all the dependencies installed. Simply skip to `Running Python Tests`_. If neither 
of those options appeals, read on.

You'll need to install tox, which is used to setup and managed the virtualenvs for the tests.

If you don't want to install that directly on your system, I suggest making a virtualenv. For example:

.. code-block:: bash

    mkvirtualenv babymailgun

Then:

.. code-block:: bash

    pip install tox

====================
Running Python Tests
====================

.. code-block:: bash

    make python_tests


========================
Running Functional Tests
========================

From your clone root:

.. code-block:: bash

    make functional_tests

===========================
Manually Testing The Worker
===========================

Mailhog exposes some lovely levers for testing out the robustness of your SMTP client implementation. I've added a commented
out section in the docker-compose.yml file called "failhog" for trying some of them out. See the `Jim <https://github.com/mailhog/MailHog/blob/master/docs/JIM.md>`_ documentation on the Mailhog site for more details

===========
The Project
===========

Consists of three primary components

- API
- CLI
- Worker

Additionally, the project relies on the following technologies and tools:

- Docker CE
- Docker Compose
- MongoDB
- Go 1.8
- Python 3.6.0
- MailHog
- Tox
- Pyenv (optional)

API
===

The API is written in Python 3, specifically targeting 3.6. It exposes multiple endpoints, allowing end-users to send emails and retrieve information about them and their sending status

CLI
===

A command line tool also written in Python 3.6, using python Click. It provides easy access to the API

Worker
======

The worker does all the heavy lifting of sending emails asynchronously, and is written against Go 1.8. It leverages goroutines to increase email throughput and interacts with MongoDB using Mongo's consistency and locking semantics, ensuring emails are only ever seen (and thus sent) by one goroutine at a time.

=====================
My Testing Philosophy
=====================

While I think there's a place for unit tests, there wasn't a lot of room for them to prove anything in this project. Put differently. unit tests are great for proving that elaborate logic is correct. What it's not great at is fetching values from the database or telling a server to manipulate
bits over the wire. In those scenarios (at least if you're truly writing unit tests) you're left asserting that your mocking skills are up to task and that some third party library actually does what it says.

With that said, I unit tested as much of the Python code as made sense to me. That includes some of the helper functions in the API code and error handling logic in the API client.

Beyond that, I've opted to provide a functional testing suite instead that makes calls to a live API and actually interacts with MongoDB and asserts that those requests and responses make sense.
