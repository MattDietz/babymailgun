FROM python:3.6.2-alpine3.6 as lifeloopweb_build_base

RUN apk update && apk upgrade && apk --update add openssl git

WORKDIR /code

RUN export DOCKERIZE_VERSION=v0.3.0 && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
        && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

ADD ./requirements.txt /code
ADD ./MANIFEST.in /code
ADD ./setup.py /code
ADD ./babymailgun /code/babymailgun

RUN pip install -r requirements.txt
RUN pip install -e .

CMD ["flask", "run"]
