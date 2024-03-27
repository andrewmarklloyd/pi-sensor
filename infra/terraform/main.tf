resource "digitalocean_droplet" "mqtt_server" {
  image  = "ubuntu-22-04-x64"
  name   = "mqtt-server"
  region = "sfo3"
  monitoring = true
  # is this the smallest size?
  size   = "s-1vcpu-1gb"
  ssh_keys = [data.digitalocean_ssh_key.do.id]

  user_data = data.local_file.userdata.content
  tags = ["mqtt-server"]
}

resource "digitalocean_firewall" "mqtt_server" {
  name = "mqtt-server"
  tags = ["mqtt-server"]

  droplet_ids = [digitalocean_droplet.mqtt_server.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = [var.ssh_inbound_ip]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "1883"
    source_addresses = [var.ssh_inbound_ip]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "9001"
    source_addresses = [var.ssh_inbound_ip]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0"]
  }
}

resource "digitalocean_domain" "mqtt" {
  name       = var.mosquitto_domain
  ip_address = digitalocean_droplet.mqtt_server.ipv4_address
}

resource "digitalocean_record" "mqtt" {
  domain = digitalocean_domain.mqtt.id
  type   = "A"
  name   = "mqtt"
  value  = digitalocean_droplet.mqtt_server.ipv4_address
}

output "ip_address" {
  value = digitalocean_droplet.mqtt_server.ipv4_address
}

output "droplet_id" {
  value = digitalocean_droplet.mqtt_server.id
}

data "local_file" "userdata" {
  filename = "./userdata.sh"
}

data "digitalocean_ssh_key" "do" {
  name = "DO SSH Key"
}
