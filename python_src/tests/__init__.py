import contextlib

import mock

import pytest


class TestBase(object):
    @contextlib.contextmanager
    def not_raises(self):
        try:
            yield
        except Exception as e:
            pytest.fail("Test incorrectly raised! Original exception: "
                        "{}".format(e))
