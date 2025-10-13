import paho.mqtt.client as mqtt


CLIENT_ID = 'detection-service'

# client = mqtt.Client()
# client.on_connect = on_connect
# client.connect("localhost", 1883, 60)  # Connect to a local broker on port 1883
#
# client.publish("my/topic", "Hello from Python!")  # Publish a message
# client.loop_start()  # Start the loop in a separate thread
# # ... do other things ...
# client.loop_stop()  # Stop the loop when done
# client.disconnect()


def get_client() -> mqtt.Client:
    c = mqtt.Client(
        client_id=CLIENT_ID,
        callback_api_version=mqtt.CallbackAPIVersion.VERSION2
    )
    return c
