#!/bin/bash

# https://github.com/prasmussen/gdrive


delete_extra_rows() {
    # COPY command is not supported with Heroku Postgres, must use psql \copy
    setupQuery="CREATE TABLE IF NOT EXISTS tmp_status (source text, status text, timestamp text);
TRUNCATE ONLY tmp_status;"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${setupQuery}"

    copyQuery="\copy tmp_status FROM '/tmp/out.csv' WITH (FORMAT CSV);"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${copyQuery}"

    delQuery="DELETE FROM status s
USING tmp_status
WHERE s.timestamp = tmp_status.timestamp;
DROP TABLE tmp_status;"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${delQuery}"
}

# Find out how many rows the database is above the defined limit. Take that number of rows, append it to the backup file, and delete from the database. Essentially treats the active db as a very large queue, when queue size is too large we pop the old and move into cold storage
limit_data_size() {
    max=${1}
    if [[ -z ${max} ]]; then
        echo "Max argument not set"
        exit 1
    fi
    list=$(gdrive list -q "name contains 'pi-sensor'")
    dir=$(echo "${list}" | grep pi-sensor)
    if [[ -z ${dir} ]]; then
        echo "Cold storage bucket does not exist, creating it now"
        gdrive upload -r ${syncDir}
        list=$(gdrive list -q "name contains 'pi-sensor'")
        dir=$(echo "${list}" | grep pi-sensor)
        export DRIVE_DIR=$(echo $dir | awk '{print $1}')
        gdrive sync upload ${syncDir} ${DRIVE_DIR}
        echo "Cold storage bucket id first 5 characters: $(echo $DRIVE_DIR| cut -c1-5)..."
    else
        export DRIVE_DIR=$(echo $dir | awk '{print $1}')
        echo "Cold storage bucket already exists, first 5 characters $(echo $DRIVE_DIR| cut -c1-5)..."
        gdrive sync download ${DRIVE_DIR} ${syncDir}
    fi
    query="SELECT COUNT(*) FROM status;"
    rowCount=$(docker run -v "${PWD}/tmp:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}")
    rowCount=$(echo "$rowCount" | tr -d '[:space:]')
    if [[ ${rowCount} -le ${max} ]]; then
        echo "Row count: '${rowCount}' is less than or equal to max: '${max}', no action required"
        return 0
    fi
    echo "Row count: '${rowCount}' is greater than max: '${max}', trimming database and syncing backup file"
    rowsAboveMax=$((${rowCount}-${max}))
    echo "Number of rows above max: ${rowsAboveMax}"
    latest=$(tail -n 1 ${syncDir}/cold-storage.csv || echo '')
    query="\copy (SELECT * FROM status ORDER by timestamp ASC LIMIT ${rowsAboveMax}) to '/tmp/out.csv' with delimiter as ','"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
    if [[ -z ${latest} ]]; then
        echo "Backup file not found, dumping full contents of query to backup file"
        cp ${tmpWorkDir}/out.csv ${syncDir}/cold-storage.csv
    else
        # Get all rows from query that are not currently in cold-storage and append to cold-storage
        awk "/${latest}/{y=1;next}y" ${tmpWorkDir}/out.csv >> ${syncDir}/cold-storage.csv
    fi
    # ensure list is sorted and unique before uploading
    sort -u -k 3 -t ',' -o ${syncDir}/cold-storage.csv ${syncDir}/cold-storage.csv
    gdrive sync upload ${syncDir} ${DRIVE_DIR} || exit 1
    # Delete rows limiting to rowsAboveMax; but ONLY if successfully uploaded to storage!!
    delete_extra_rows
}

export DATABASE_URL=$(heroku config:get DATABASE_URL -a pi-sensor-staging)
eval `~/parse-posgres-url.js`
syncDir='/tmp/pi-sensor'
rm -rf ${syncDir}
mkdir -p ${syncDir}
tmpWorkDir='/tmp/data'
rm -rf ${tmpWorkDir}
mkdir -p ${tmpWorkDir}
MAX_ROWS=10

limit_data_size ${MAX_ROWS}
