resource "digitalocean_droplet" "pi_sensor_data" {
  image  = "ubuntu-22-04-x64"
  name   = "pi-sensor-data"
  region = "sfo3"
  monitoring = true
  # is this the smallest size?
  size   = "s-1vcpu-1gb"
  ssh_keys = [data.digitalocean_ssh_key.do.id]

  user_data = data.local_file.userdata.content
  tags = ["mqtt-server"]
}

# need UDP 41641

output "ip_address" {
  value = digitalocean_droplet.pi_sensor_data.ipv4_address
}

output "droplet_id" {
  value = digitalocean_droplet.pi_sensor_data.id
}

data "local_file" "userdata" {
  filename = "./userdata.sh"
}

data "digitalocean_ssh_key" "do" {
  name = "DO SSH Key"
}
