function translateStatus(status) {
  var icon, color
  if (status === "OPEN") {
    icon = "unlock"
    color = "red"
  } else if (status === "CLOSED") {
    icon = "lock"
    color = "green"
  } else {
    icon = "zap-off"
    color = "grey"
  }
  return {
      color,
      icon
  }
}

function unixToDate(unixTimestamp) {
  var date = new Date(unixTimestamp * 1000)
  var year = date.getFullYear()
  var month = date.getMonth() + 1
  var day = date.getDate()
  var hours = date.getHours()
  var minutes = "0" + date.getMinutes()
  var seconds = "0" + date.getSeconds()

  return `${year}-${month}-${day} ${hours}:${minutes.substr(-2)}:${seconds.substr(-2)}`
}

function timeSince(unixTimestamp) {
  var date = new Date(unixTimestamp * 1000)
  var seconds = Math.floor((new Date() - date) / 1000)
  var interval = seconds / 31536000
  if (interval > 1) {
    return Math.floor(interval) + " years ago"
  }
  interval = seconds / 2592000
  if (interval > 1) {
    return Math.floor(interval) + " months ago"
  }
  interval = seconds / 86400
  if (interval > 1) {
    return Math.floor(interval) + " days ago"
  }
  interval = seconds / 3600
  if (interval > 1) {
    return Math.floor(interval) + " hours ago"
  }
  interval = seconds / 60
  if (interval > 1) {
    return Math.floor(interval) + " minutes ago"
  }
  return Math.floor(seconds) + " seconds ago"
}

function trimVersion(version) {
  if (version === "") {
    return version
  }
  return version.substring(0,7)
}

export {translateStatus, unixToDate, timeSince, trimVersion}
