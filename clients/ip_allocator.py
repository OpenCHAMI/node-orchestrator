import requests
import ipaddress

# Function to get the payload from the server
def get_payload(url):
    response = requests.get(url)
    response.raise_for_status()  # Ensure we notice bad responses
    return response.json()

# Function to find the next available IP address in the subnet
def get_next_available_ip(assigned_ips, subnet='192.168.1.0/24'):
    network = ipaddress.ip_network(subnet)
    for ip in network.hosts():
        if str(ip) not in assigned_ips:
            return str(ip)
    raise RuntimeError("No available IPs in the subnet")

# Function to update nodes with new IPs for eth0
def update_nodes(nodes, url):
    assigned_ips = {iface['mac_address']: iface['ip_address'] 
                    for node in nodes 
                    for iface in node['network_interfaces'] 
                    if 'ip_address' in iface and iface['ip_address']}
    
    for node in nodes:
        for iface in node['network_interfaces']:
            if iface['interface_name'] == 'eth0' and 'ip_address' not in iface:
                new_ip = get_next_available_ip(assigned_ips)
                iface['ip_address'] = new_ip
                assigned_ips[iface['mac_address']] = new_ip
                response = requests.post(url+"/"+node['id'], json=node)
                response.raise_for_status()  # Ensure we notice bad responses

# Main function to orchestrate the tasks
def main():
    base_url = "http://localhost:8080/ComputeNode"
    payload = get_payload(base_url)
    
    nodes_with_eth0_no_ip = [
        node for node in payload if 'network_interfaces' in node and any(
            iface['interface_name'] == 'eth0' and 'ip_address' not in iface
            for iface in node['network_interfaces']
        )]
    
    print(len(nodes_with_eth0_no_ip))
    update_nodes(nodes_with_eth0_no_ip, base_url)

if __name__ == "__main__":
    main()