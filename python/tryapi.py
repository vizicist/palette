import asyncio
import nats
from nats.errors import ConnectionClosedError, TimeoutError, NoServersError

async def main():

    nc = await nats.connect("nats://127.0.0.1:4222")

    try:
        response = await nc.request("toengine.api", b'{ "api":"engine.status" }', timeout=0.5)
        print("Received response: {message}".format(
            message=response.data.decode()))
    except TimeoutError:
        print("Request timed out")

    # Terminate connection to NATS.
    print("NEED TO DRAIN?")
    # await nc.drain()

if __name__ == '__main__':
    asyncio.run(main())
