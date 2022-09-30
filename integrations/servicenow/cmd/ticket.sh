#!/bin/bash

SNOW_CLIENT_BINARY=${HOME}/bin/snow_client

if [ "${_PLUGINNAME}" = "FKM" ]
then
ID="${_GATEWAY}${_NETPROBE_HOST}${_MANAGED_ENTITY}${_SAMPLER}${_DATAVIEW}${_COLUMN}${_triggerDetails}${_Filename}"
else
ID="${_GATEWAY}${_NETPROBE_HOST}${_MANAGED_ENTITY}${_SAMPLER}${_DATAVIEW}${_ROWNAME}${_COLUMN}${_HEADLINE}"
fi

SHORT="Incident in ${REGION}: Value ${_MANAGED_ENTITY} | ${_SAMPLER} | ${_ROWNAME} | ${_COLUMN}${_HEADLINE}"
TEXT="Geneos time: ${_ALERT_CREATED}\nGateway: ${_GATEWAY}\nManaged Entity: ${_MANAGED_ENTITY}\nPlugin: ${_PLUGINNAME}\nSampler: ${_SAMPLER}\nRow: ${_ROWNAME}\nColumn/headline: ${_COLUMN}${_HEADLINE}\nValue: ${_VALUE}${_triggerDetails}\nEnvironment: ${ENVIRONMENT}\nClient: ${CLIENT}\nRegion: ${Region}\nLocation: ${LOCATION}\nPlatform: ${PLATFORM}\nComponent: ${COMPONENT}"
SEARCH="name=${_NETPROBE_HOST}"
# SEARCH="sys_id=${UUID}"

${SNOW_CLIENT_BINARY} --short "${SHORT}" --search "${SEARCH}" --text "${TEXT}" --id "${ID:-$_MANAGED_ENTITY}" --severity "${_SEVERITY}" category="${COMPONENT:-Hardware}"
