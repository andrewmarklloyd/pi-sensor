#!/bin/bash


get_all() {
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    query="\copy (SELECT * FROM status ORDER by timestamp ASC) to '/tmp/backup-full.csv' with delimiter as ','"
    docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
}

limit_data_size() {
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    max=400
    query="SELECT COUNT(*) FROM status"
    rowCount=$(docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}")
    rowCount=$(echo "$rowCount" | tr -d '[:space:]')
    if [[ ${rowCount} -le ${max} ]]; then
        echo "Row count: '${rowCount}' less than or equal to max: '${max}', no action required"
        return 0
    fi
    rowsAboveMax=$((${rowCount}-${max}))
    echo "Number of rows above max: ${rowsAboveMax}"
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    query="\copy (SELECT * FROM status ORDER by timestamp ASC LIMIT ${rowsAboveMax}) to '/tmp/out.csv' with delimiter as ','"
    docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
    # TODO: copy file from object storage, output to tmp/backup-full.csv
    if [[ -f ~/Desktop/backup-full.csv ]]; then
        cp ~/Desktop/backup-full.csv tmp/backup-full.csv
    fi
    latest=$(tail -n 1 tmp/backup-full.csv || echo '')
    if [[ -z ${latest} ]]; then
        echo "Backup file not found, dumping contents of query to tmp/out.csv to backup file"
        cp tmp/out.csv tmp/backup-full.csv
    else
        awk "/${latest}/{y=1;next}y" tmp/out.csv >> tmp/backup-full.csv
    fi
    # TODO: copy tmp/backup-full.csv to object storage
    cp tmp/backup-full.csv ~/Desktop/
}

limit_data_size
