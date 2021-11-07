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
    bucket=${2}
    if [[ -z ${max} ]]; then
        echo "Max argument not set"
        exit 1
    fi
    if [[ -z ${bucket} ]]; then
        echo "Bucket argument not set"
        exit 1
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
    query="\copy (SELECT * FROM status ORDER by timestamp ASC LIMIT ${rowsAboveMax}) to '/tmp/out.csv' with delimiter as ','"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"

    query="\copy (SELECT * FROM status) to '/tmp/full-before.csv' with delimiter as ','"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"

    cat ${tmpWorkDir}/out.csv >> ${syncDir}/cold-storage.csv
    # ensure list is sorted and unique before uploading
    sort -u -k 3 -t ',' -o ${syncDir}/cold-storage.csv ${syncDir}/cold-storage.csv
    gdrive sync upload ${syncDir} ${bucket} || exit 1
    # Delete rows limiting to rowsAboveMax; but ONLY if successfully uploaded to storage!!
    delete_extra_rows
    query="\copy (SELECT * FROM status) to '/tmp/full-after.csv' with delimiter as ','"
    docker run -v "${tmpWorkDir}:/tmp" -e PGPASSWORD=${pw} -it --rm postgres psql -h ${host} -U ${user} ${db} -t -c "${query}"
}

setup_bucket() {
    bucketName=${1}
    syncDir=${2}
    if [[ -z ${bucketName} ]]; then
        echo "Bucket name argument not set"
        exit 1
    fi
    if [[ -z ${syncDir} ]]; then
        echo "Sync directory argument not set"
        exit 1
    fi
    bucketList=$(gdrive list -q "name = '${bucketName}'")
    bucket=$(echo "${bucketList}" | grep ${bucketName} | awk '{print $1}')
    if [[ -z ${bucket} ]]; then
        echo "Cold storage bucket does not exist, creating it now"
        gdrive upload -r ${syncDir}
        bucketList=$(gdrive list -q "name = '${bucketName}'")
        bucket=$(echo "${bucketList}" | grep ${bucketName} | awk '{print $1}')
        gdrive sync upload ${syncDir} ${bucket}
        export DRIVE_DIR=${bucket}
        echo "Cold storage bucket id first 5 characters: $(echo $DRIVE_DIR| cut -c1-5)..."
    else
        echo "Cold storage bucket already exists, first 5 characters $(echo $bucket| cut -c1-5)..."
        gdrive sync download ${bucket} ${syncDir}
    fi
    echo ${bucket} > /tmp/.bucket
}

app=pi-sensor-staging
bucketName="backup-${app}"
MAX_ROWS=10

export DATABASE_URL=$(heroku config:get DATABASE_URL -a ${app})
eval `~/parse-posgres-url.js`
syncDir="/tmp/${bucketName}"
rm -rf ${syncDir}
mkdir -p ${syncDir}
tmpWorkDir='/tmp/data'
rm -rf ${tmpWorkDir}
mkdir -p ${tmpWorkDir}

setup_bucket ${bucketName} ${syncDir}
bucket=$(cat /tmp/.bucket)

limit_data_size ${MAX_ROWS} ${bucket}
