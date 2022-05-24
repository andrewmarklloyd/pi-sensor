#!/bin/bash

set -eu

redis() {
    REDIS_URL=$(heroku config:get REDIS_URL -a pi-sensor)

    tmp=${REDIS_URL#*//:}
    pw=${tmp%@*}
    hostTmp=${tmp#*@}
    host=${hostTmp%:*}
    port=${hostTmp#*:}

    redis-cli -a ${pw} -h ${host} -p ${port}
}

aws_commands() {
    aws s3api put-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME} --versioning-configuration Status=Enabled

    cat /tmp/pi-sensor-staging | jq -c -s 'unique_by(.timestamp)[]' > /tmp/pi-sensor-staging-2
    aws s3 cp /tmp/pi-sensor-staging-2 s3://${BUCKETEER_BUCKET_NAME}/backups/pi-sensor-staging
}

full_backup_s3() {
  f="pi-sensor-$(date +%s).json"
  rm -f /tmp/pi-sensor
  go run main.go
  mv /tmp/pi-sensor ${f}
  aws s3 cp ${f} s3://${BUCKETEER_BUCKET_NAME}/backups/${f}
}

restore_local_db() {
    export PGHOST=localhost
    export PGPORT=5432
    export PGUSER=postgres
    export PGPASSWORD=mysecretpassword
    echo postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}?sslmode=disable | pbcopy

    docker rm -f postgres

    aws s3 cp s3://${BUCKETEER_BUCKET_NAME}/backups/pi-sensor-staging /tmp/pi-sensor-staging

    docker run -it --name postgres -v /tmp:/tmp -p 5432:5432 -e POSTGRES_PASSWORD=${PGPASSWORD} -d postgres

    sleep 5

    echo 'CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text, version text);' > /tmp/tmp.sql
    jq -r -s ".[] | \"INSERT INTO status(source, status, timestamp, version) VALUES('\(.source)', '\(.status)', '\(.timestamp)', '\(.version)');\"" /tmp/pi-sensor-staging >> /tmp/tmp.sql

    psql -a -f /tmp/tmp.sql
}

mock_data() {
    rm -f /tmp/mock.sql
    for x in $(seq 1 100); do
        t=$(date -v-${x}m +%s)
        echo "INSERT INTO status(source, status, timestamp, version) VALUES('garage', 'OPEN', '${t}', '1671a0a8c76461d43763e67b503756f8ed685c7c');" >> /tmp/mock.sql
    done
    psql ${DATABASE_URL} -a -f /tmp/mock.sql
}

mock_data
