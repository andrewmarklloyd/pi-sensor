import React, { Component } from "react";

import {
  Page,
  Button,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";

var vapidPublicKey

class NotificationsPage extends Component {
  constructor(props) {
    super(props)
    vapidPublicKey = process.env.REACT_APP_VAPID_PUBLIC_KEY

    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('service-worker.js')
    }
  }
  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Button color="secondary" onClick={() => subscribe()}>Subscribe</Button>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

function subscribe() {
  if (!("Notification" in window) || !('serviceWorker' in navigator)) {
    alert("This browser does not support desktop notification")
    return
  }

  if (Notification.permission === "granted") {
    createOrUpdateSubscription()
    return
  }

  if (Notification.permission !== "denied") {
    Notification.requestPermission().then((permission) => {
      if (permission === "granted") {
        createOrUpdateSubscription()
      }
    })
  } else {
    console.log('[debug] receieved unexpected response permission: ', Notification.permission)
  }
}

function createOrUpdateSubscription() {
  var pushManager
  navigator.serviceWorker.ready
    .then(function(registration) {
      pushManager = registration.pushManager
      return registration.pushManager.getSubscription()
    })
    .then(function(subscription) {
      if (subscription) {
        return subscription
      } else {
        return pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
        })
      }
    })
    .then(function(subscription) {
      fetch("/api/subscription", {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json'
        },
        referrerPolicy: 'no-referrer',
        body: JSON.stringify(subscription)
      })
      .then(r => r.json())
      .then(j => {
        if (j.status !== 'success') {
          alert("error subscribing to notifications: " + j.error)
        } else {
          alert("successfully subscribed to notifications")
        }
      })
    })
    .catch(err => {
      alert("error subscribing to notifications")
    });
}

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');
  const rawData = window.atob(base64);
  return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
}

export default NotificationsPage;
