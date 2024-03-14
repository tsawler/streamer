# Streamer

## key info
Generate a key and vector:

```
openssl rand 16 > enc.key  # Key to encrypt the video
openssl rand -hex 16       # IV
# de0efc88a53c730aa764648e545e3874
```

Create enc.keyinfo:
```
https://whatever.com/enc.key
enc.key
de0efc88a53c730aa764648e545e3874
```

*Note*: enc.key **must** be at the root level of the project that uses this
package. The location of enc.keyinfo can be specified when creating
a `streamer.VideoProcessor` object.