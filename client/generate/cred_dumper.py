
from __future__ import print_function

import awscli.clidriver
import json
import sys

driver = awscli.clidriver.create_clidriver()
table = driver._get_command_table()
parser = driver._create_parser(table)
args, _ = parser.parse_known_args(["--debug", "s3", "ls"])

driver._emit_session_event(args)

session = driver.session

creds = session.get_credentials()

if creds is None:
    sys.exit(1)

print(json.dumps({
    "AWS_ACCESS_KEY_ID": creds.access_key,
    "AWS_SECRET_ACCESS_KEY": creds.secret_key,
    "AWS_SESSION_TOKEN": creds.token,
    "AWS_REGION": session.get_scoped_config()["region"],
}))
