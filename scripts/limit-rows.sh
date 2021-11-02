#!/bin/bash


limit_data_size() {
    # copy file from object storage, output to tmp/storage.csv
    latest=$(tail -n 1 tmp/storage.csv || echo '')
    if [[ -z ${latest} ]]; then
        limitTs='1609491600' # 2021-01-01
    else
        arrIN=(${latest//,/ })
        limitTs=${arrIN[2]}
    fi
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    limit=100
    query="\copy (SELECT * FROM status WHERE timestamp > '${limitTs}' ORDER by timestamp ASC LIMIT ${limit}) to '/tmp/out.csv' with delimiter as ','"
    docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -c "${query}"
    cat tmp/out.csv >> tmp/storage.csv
    # copy tmp/storage.csv to object storage
}

limit_data_size
