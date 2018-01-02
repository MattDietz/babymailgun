#!/bin/sh

python add_server.py
dockerize -timeout 60s -wait tcp://database:27017 flask run -h 0.0.0.0 -p5000 --reload
