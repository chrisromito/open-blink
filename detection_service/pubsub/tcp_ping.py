import socket


def ping(host, port, timeout: int = 1) -> bool:
    """
    Attempts to establish a TCP connection to a host and port.

    Args:
        host (str): The target IP address or hostname.
        port (int): The target TCP port number.
        timeout (int): The timeout in seconds for the connection attempt.

    Returns:
        bool: True if the connection is successful, False otherwise.
    """

    # Create a socket object
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.settimeout(timeout)
        # Attempt to connect
        s.connect((host, port))
        print(f"Successfully connected to {host}:{port}")
        s.close()
        return True
    except socket.error as e:
        s.close()
        print(f"Failed to connect to {host}:{port}. Error: {e}")
        return False
