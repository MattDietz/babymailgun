#!/usr/bin/env python

# Setup.py egregiously copied from another project I maintain
import sys
import distutils.core

from pip.download import PipSession
from pip.req import parse_requirements

__version__ = "0.1"

try:
    import setuptools
except ImportError:
    pass


def requires(path):
    return [str(r.req) for r in parse_requirements(path, session=PipSession())
            if r]

distutils.core.setup(
    name="babymailgun",
    version=__version__,
    packages=["babymailgun"],
    package_data={
        "babymailgun": [],
        },
    author="Matt Dietz",
    author_email="matthew.dietz@gmail.com",
    url="https://github.com/cerberus98/babymailgun",
    download_url="https://github.com/cerberus98/babymailgun",
    license="https://github.com/cerberus98/babymailgun/blob/master/LICENSE",
    description="",
    install_requires=requires("requirements.txt"),
    entry_points={
        "console_scripts": [
            "mailgun_cli = babymailgun.shell:main",
        ]}
    )
