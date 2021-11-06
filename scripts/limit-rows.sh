#!/bin/bash

# https://github.com/prasmussen/gdrive


get_all() {
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    query="\copy (SELECT * FROM status ORDER by timestamp ASC) to '/tmp/backup-full.csv' with delimiter as ','"
    docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
}

limit_data_size() {
    syncDir='/tmp/pi-sensor'
    rm -rf ${syncDir}
    mkdir -p ${syncDir}
    tmpWorkDir='/tmp/data'
    rm -rf ${tmpWorkDir}
    mkdir -p ${tmpWorkDir}
    list=$(gdrive list -q "name contains 'pi-sensor'")
    dir=$(echo "${list}" | grep pi-sensor)
    if [[ -z ${dir} ]]; then
        echo "Backup object storage does not exist, creating it now"
        # echo keep > ${syncDir}/.keep
        gdrive upload -r ${syncDir}
        list=$(gdrive list -q "name contains 'pi-sensor'")
        dir=$(echo "${list}" | grep pi-sensor)
        export DRIVE_DIR=$(echo $dir | awk '{print $1}')
        gdrive sync upload ${syncDir} ${DRIVE_DIR}
        echo "Backup object storage id: ${DRIVE_DIR}"
    else
        export DRIVE_DIR=$(echo $dir | awk '{print $1}')
        echo "Backup object storage already exists ${DRIVE_DIR}"
        gdrive sync download ${DRIVE_DIR} ${syncDir}
    fi
    export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
    eval `~/parse-posgres-url.js`
    max=400
    query="SELECT COUNT(*) FROM status"
    rowCount=$(docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}")
    rowCount=$(echo "$rowCount" | tr -d '[:space:]')
    if [[ ${rowCount} -le ${max} ]]; then
        echo "Row count: '${rowCount}' is less than or equal to max: '${max}', no action required"
        return 0
    fi
    echo "Row count: '${rowCount}' is greater than max: '${max}', trimming database and syncing backup file"
    rowsAboveMax=$((${rowCount}-${max}))
    echo "Number of rows above max: ${rowsAboveMax}"
    latest=$(tail -n 1 ${syncDir}/backup-full.csv || echo '')
    query="\copy (SELECT * FROM status ORDER by timestamp ASC LIMIT ${rowsAboveMax}) to '/tmp/out.csv' with delimiter as ','"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
    if [[ -z ${latest} ]]; then
        echo "Backup file not found, dumping contents of query to backup file"
        cp ${tmpWorkDir}/out.csv ${syncDir}/backup-full.csv
    else
        awk "/${latest}/{y=1;next}y" ${tmpWorkDir}/out.csv >> ${syncDir}/backup-full.csv
    fi
    sort -u -k 3 -t ',' -o ${syncDir}/backup-full.csv ${syncDir}/backup-full.csv
    gdrive sync upload ${syncDir} ${DRIVE_DIR}
    # Delete rows limiting to rowsAboveMax; but ONLY if successfully uploaded to storage!!
}

limit_data_size
