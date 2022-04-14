#!/bin/bash

set -euo pipefail

app=${1}

heroku container:login
heroku container:push web -a ${app}
heroku container:release web -a ${app}
curl https://${app}.herokuapp.com/health
