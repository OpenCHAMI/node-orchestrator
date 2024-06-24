import json
import uuid
import random
import sys
from faker import Faker

fake = Faker()

def generate_compute_node():
    num_interfaces = random.randint(1, 4)
    network_interfaces = []
    for _ in range(num_interfaces):
        interface = {
            "interface_name": random.choice(["eth0", "eth1", "ib0", "ib1", "ip2", "ip3"]),
            "mac_address": fake.mac_address(),
            "description": fake.sentence()
        }
        network_interfaces.append(interface)
    
    has_bmc = random.choices([True, False], weights=[0.4, 0.6])[0]
    bmc = None
    if has_bmc:
        bmc = {
            "username": "admin",
            "password": "admin",
            "mac_address": fake.mac_address()
        }
    
    payload = {
        "hostname": fake.hostname(0),
        "xname": f"x{random.randint(10000, 99999)}c{random.randint(1, 60)}s{random.randint(1, 10)}b{random.randint(1, 3)}n{random.randint(1, 8)}",
        "architecture": random.choice(["x86_64", "arm64"]),
        "boot_mac": fake.mac_address(),
        "network_interfaces": network_interfaces,
        "description": fake.sentence(),
    }
    if bmc is not None:
        payload["bmc"] = bmc
    return payload

def main():
    if len(sys.argv) != 2:
        print("Usage: python gen_fake_data.py <number_of_nodes>")
        sys.exit(1)

    num_nodes = int(sys.argv[1])
    compute_nodes = [generate_compute_node() for _ in range(num_nodes)]

    with open('fake_compute_nodes.json', 'w') as f:
        json.dump(compute_nodes, f, indent=4)

if __name__ == "__main__":
    main()