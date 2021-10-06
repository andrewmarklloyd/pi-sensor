#!/bin/bash


balance=$(curl -s https://api.twilio.com/2010-04-01/Accounts/${TWILIO_ACCOUNT_SID}/Balance.json -u "${TWILIO_ACCOUNT_SID}:${TWILIO_AUTH_TOKEN}" | /app/jq -r .balance)
echo "Current balance: ${balance}"
limit=0.5
if (( $(echo "${balance} ${limit}" | awk '{print ($1 < $2)}') )); then
  echo "Twilio balance almost depleted"
fi
