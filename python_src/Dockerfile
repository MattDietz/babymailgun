FROM python:3.6.2-alpine3.6 as lifeloopweb_build_base

RUN apk update && apk upgrade && apk --update add openssl git

WORKDIR /code

RUN export DOCKERIZE_VERSION=v0.3.0 && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
        && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# Adding all of these individually is safer than accidentally copying python build artifacts
# into the container
ADD ./requirements.txt /code
ADD ./test-requirements.txt /code
ADD ./tox.ini /code
ADD ./tests /code/tests
ADD ./.pylintrc /code
ADD ./docker/api_entrypoint.sh /code/api_entrypoint.sh
ADD ./docker/add_server.py /code/add_server.py
ADD ./MANIFEST.in /code
ADD ./setup.py /code
ADD ./babymailgun /code/babymailgun

# These pycache dirs cause havoc with running stuff in the container
RUN find . -name "*pycache*" -exec rm -rf {} \;

RUN pip install -r requirements.txt
RUN pip install -r test-requirements.txt
RUN pip install -e .

CMD ["flask", "run"]
