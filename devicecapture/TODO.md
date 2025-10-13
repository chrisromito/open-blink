- device
  - /somewhere - Implement struct Frame { Image: image.Image, Timestamp: int }
  - /api.go - Put all available API calls here

- pubsub
  - client - Handle multiple broker URLS
  - Allow publishing images to topic: `image/{DEVICE_ID}`