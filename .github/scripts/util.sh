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

aws_bucket_config() {
    aws s3api put-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME} --versioning-configuration Status=Enabled
    aws s3api get-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME}
    aws s3api put-bucket-lifecycle-configuration \
        --bucket ${BUCKETEER_BUCKET_NAME} \
        --lifecycle-configuration file://assets/lifecycle.json
    aws s3api get-bucket-lifecycle-configuration --bucket ${BUCKETEER_BUCKET_NAME}
}

aws_commands() {
    aws s3 ls ${BUCKETEER_BUCKET_NAME} --recursive --human-readable --summarize
    cat /tmp/backups/pi-sensor-full-backup.json | jq -c -s 'unique_by(.timestamp)[]' > /tmp/backups/pi-sensor-full-backup.json-2
    aws s3 cp /tmp/backups/pi-sensor-full-backup.json-2 s3://${BUCKETEER_BUCKET_NAME}/backups/backups/pi-sensor-full-backup.json
}

restore_local_db() {
    export PGHOST=localhost
    export PGPORT=5432
    export PGUSER=postgres
    export PGPASSWORD=mysecretpassword
    echo postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}?sslmode=disable | pbcopy

    docker rm -f postgres

    aws s3 cp s3://${BUCKETEER_BUCKET_NAME}/backups/pi-sensor-full-backup.json /tmp/backups/pi-sensor-full-backup.json

    docker run -it --name postgres -v /tmp:/tmp -p 5432:5432 -e POSTGRES_PASSWORD=${PGPASSWORD} -d postgres

    sleep 5

    echo 'CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text, version text);' > /tmp/tmp.sql
    jq -r -s ".[] | \"INSERT INTO status(source, status, timestamp, version) VALUES('\(.source)', '\(.status)', '\(.timestamp)', '\(.version)');\"" /tmp/backups/pi-sensor-full-backup.json >> /tmp/tmp.sql

    psql -a -f /tmp/tmp.sql
}

mock_data() {
    rm -f /tmp/mock.sql
    for x in $(seq 1 50); do
        t=$(date -v-${x}m +%s)
        echo "INSERT INTO status(source, status, timestamp, version) VALUES('garage', 'OPEN', '${t}', '1671a0a8c76461d43763e67b503756f8ed685c7c');" >> /tmp/mock.sql
    done
    psql ${DATABASE_URL} -a -f /tmp/mock.sql
}

get_config() {
    app=${1}
    heroku config -a ${app} -j | jq -r 'to_entries[] | "export \(.key)=\(.value)"'
}

