import asyncio
import nats
from nats.errors import ConnectionClosedError, TimeoutError, NoServersError

async def main():
    # It is very likely that the demo server will see traffic from clients other than yours.
    # To avoid this, start your own locally and modify the example to use it.
    nc = await nats.connect("nats://127.0.0.1:4222")

    async def message_handler(msg):
        subject = msg.subject
        reply = msg.reply
        data = msg.data.decode()
        print("Received a message on '{subject} {reply}': {data}".format(
            subject=subject, reply=reply, data=data))

    # Simple publisher and async subscriber via coroutine.
    sub = await nc.subscribe(">", cb=message_handler)

    try:
        async for msg in sub.messages:
            print(f"Received a message on '{msg.subject} {msg.reply}': {msg.data.decode()}")
            await sub.unsubscribe()
    except Exception as e:
        pass

    async def help_request(msg):
        print(f"Received a message on '{msg.subject} {msg.reply}': {msg.data.decode()}")
        await nc.publish(msg.reply, b'I can help')

    # Send a request and expect a single response
    # and trigger timeout if not faster than 500 ms.
    try:
        response = await nc.request("toengine.api", b'{ "api":"engine.status" }', timeout=0.5)
        print("Received response: {message}".format(
            message=response.data.decode()))
    except TimeoutError:
        print("Request timed out")

    # Remove interest in subscription.
    await sub.unsubscribe()

    # Terminate connection to NATS.
    print("NEED TO DRAIN?")
    # await nc.drain()

if __name__ == '__main__':
    asyncio.run(main())
