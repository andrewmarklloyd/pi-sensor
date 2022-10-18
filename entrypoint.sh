#!/bin/sh


list_firewalls() {
    curl -s -X GET \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $DO_ACCESS_TOKEN" \
        "https://api.digitalocean.com/v2/firewalls"
}

get_firewall() {
    id=${1}
    curl -s -X GET \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $DO_ACCESS_TOKEN" \
        "https://api.digitalocean.com/v2/firewalls/${id}"
}

configure() {
    IP=$(curl -s ifconfig.me)
    firewalls=$(list_firewalls)
    id=$(echo ${firewalls} | jq -r ".firewalls[] | select(.name == \"${FIREWALL_NAME}\").id")
    firewall=$(get_firewall ${id})

    existing=$(echo ${firewall} | jq --arg ip ${IP} '.firewall.inbound_rules[] | select(.ports == "8080")')
    tmp=$(echo ${existing} | jq "del(.sources.addresses[] | select(. != \"${STATIC_INBOUND_IP}\"))")
    updatedRules=$(echo $tmp | jq ".sources.addresses += [\"${IP}\"]" | jq -s)
    newFirewall=$(echo ${firewall} | jq --arg name "${FIREWALL_NAME}" --argjson rules "${updatedRules}" '.firewall.inbound_rules=$rules | .firewall')
    
    curl -s -X PUT \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${DO_ACCESS_TOKEN}" \
        -d "${newFirewall}" \
        "https://api.digitalocean.com/v2/firewalls/${id}" >/dev/null
}


if [ ${RUNTIME} == "D_O" ]; then
    configure
fi

unset DO_TOKEN
unset DO_FIREWALL_ID

/app/pi-sensor-server