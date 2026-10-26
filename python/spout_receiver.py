"""Display a Resolume Spout sender in a borderless GLFW window (Windows).

Install with 64-bit Python 3.13 (3.14 has no published SpoutGL wheel):
    py -3.13 -m pip install -r python/requirements-spout.txt

Run from the Palette project directory:
    py -3.13 python/spout_receiver.py --list
    py -3.13 python/spout_receiver.py --sender "Resolume Arena" \
        --x 1920 --y 0 --width 1280 --height 720 --always-on-top

Use the exact sender name shown in Resolume. The image stretches to fill the
requested window. Negative x/y coordinates support monitors left of or above
the primary display. Press Escape or Alt+F4 to close the window.

SpoutGL texture API reference:
https://github.com/jlai/Python-SpoutGL/blob/main/examples/texture/receiver.py
"""

import argparse
import sys


def positive_int(value):
    number = int(value)
    if number <= 0:
        raise argparse.ArgumentTypeError("must be greater than zero")
    return number


def parse_args(argv=None):
    parser = argparse.ArgumentParser(
        description="Display a Spout sender in a borderless window.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--list", action="store_true", help="print available Spout senders and exit"
    )
    parser.add_argument("--sender", help="exact Spout sender name (required unless --list)")
    parser.add_argument("--x", type=int, default=100, help="window left position")
    parser.add_argument("--y", type=int, default=100, help="window top position")
    parser.add_argument("--width", type=positive_int, default=1280, help="window width")
    parser.add_argument("--height", type=positive_int, default=720, help="window height")
    parser.add_argument(
        "--always-on-top", action="store_true", help="keep above regular windows"
    )
    parser.add_argument(
        "--flip-y", action="store_true", help="flip the received image vertically"
    )
    args = parser.parse_args(argv)
    if not args.list and (not args.sender or not args.sender.strip()):
        parser.error("--sender is required and must not be empty unless --list is used")
    return args


def run(args):
    # Delay optional imports so --help works before dependencies are installed.
    import SpoutGL

    if args.list:
        with SpoutGL.SpoutReceiver() as receiver:
            senders = sorted(receiver.getSenderList())
        if senders:
            for sender in senders:
                print(sender)
        else:
            print("No Spout senders found.")
        return

    import glfw
    from OpenGL import GL

    if not glfw.init():
        raise RuntimeError("GLFW initialization failed")

    window = None
    texture = 0
    try:
        # OpenGL 2.1 provides the compatibility functions used to draw the quad.
        glfw.window_hint(glfw.CONTEXT_VERSION_MAJOR, 2)
        glfw.window_hint(glfw.CONTEXT_VERSION_MINOR, 1)
        glfw.window_hint(glfw.DECORATED, glfw.FALSE)
        glfw.window_hint(
            glfw.FLOATING, glfw.TRUE if args.always_on_top else glfw.FALSE
        )
        # Position the window before showing it to avoid an initial flash elsewhere.
        glfw.window_hint(glfw.VISIBLE, glfw.FALSE)
        window = glfw.create_window(
            args.width, args.height, f"Spout: {args.sender}", None, None
        )
        if not window:
            raise RuntimeError("Could not create the OpenGL window")

        glfw.set_window_pos(window, args.x, args.y)
        glfw.make_context_current(window)
        glfw.swap_interval(1)

        def on_key(win, key, scancode, action, mods):
            if key == glfw.KEY_ESCAPE and action == glfw.PRESS:
                glfw.set_window_should_close(win, True)

        glfw.set_key_callback(window, on_key)

        GL.glActiveTexture(GL.GL_TEXTURE0)
        texture = int(GL.glGenTextures(1))
        GL.glBindTexture(GL.GL_TEXTURE_2D, texture)
        for parameter in (GL.GL_TEXTURE_MIN_FILTER, GL.GL_TEXTURE_MAG_FILTER):
            GL.glTexParameteri(GL.GL_TEXTURE_2D, parameter, GL.GL_LINEAR)
        for parameter in (GL.GL_TEXTURE_WRAP_S, GL.GL_TEXTURE_WRAP_T):
            GL.glTexParameteri(GL.GL_TEXTURE_2D, parameter, GL.GL_CLAMP_TO_EDGE)
        GL.glTexImage2D(
            GL.GL_TEXTURE_2D, 0, GL.GL_RGBA, 1, 1, 0,
            GL.GL_RGBA, GL.GL_UNSIGNED_BYTE, None,
        )
        GL.glBindTexture(GL.GL_TEXTURE_2D, 0)
        GL.glClearColor(0.0, 0.0, 0.0, 1.0)
        glfw.show_window(window)

        print(f"Waiting for Spout sender {args.sender!r}. Escape to quit.", flush=True)
        # Release Spout while its OpenGL context is still current.
        with SpoutGL.SpoutReceiver() as receiver:
            receiver.setReceiverName(args.sender)
            connected_before = False
            while not glfw.window_should_close(window):
                glfw.poll_events()
                if glfw.window_should_close(window):
                    break
                width, height = glfw.get_framebuffer_size(window)
                if width == 0 or height == 0:
                    glfw.wait_events_timeout(0.05)
                    continue

                received = receiver.receiveTexture(
                    texture, GL.GL_TEXTURE_2D, args.flip_y, 0
                )
                updated = receiver.isUpdated()
                if updated:
                    sender_width = receiver.getSenderWidth()
                    sender_height = receiver.getSenderHeight()
                    if sender_width > 0 and sender_height > 0:
                        GL.glBindTexture(GL.GL_TEXTURE_2D, texture)
                        GL.glTexImage2D(
                            GL.GL_TEXTURE_2D, 0, GL.GL_RGBA,
                            sender_width, sender_height, 0,
                            GL.GL_RGBA, GL.GL_UNSIGNED_BYTE, None,
                        )
                        GL.glBindTexture(GL.GL_TEXTURE_2D, 0)
                        print(f"Sender size: {sender_width} x {sender_height}", flush=True)
                    # The new storage will receive its first image next iteration.
                    received = False

                connected = receiver.isConnected()
                if connected != connected_before:
                    print("Connected." if connected else "Sender lost; waiting.", flush=True)
                    connected_before = connected

                GL.glViewport(0, 0, width, height)
                GL.glClear(GL.GL_COLOR_BUFFER_BIT)
                if received and connected:
                    # Match SpoutGL's example: texture (0, 0) is at the top left.
                    GL.glMatrixMode(GL.GL_PROJECTION)
                    GL.glLoadIdentity()
                    GL.glOrtho(0, 1, 1, 0, -1, 1)
                    GL.glMatrixMode(GL.GL_MODELVIEW)
                    GL.glLoadIdentity()
                    GL.glActiveTexture(GL.GL_TEXTURE0)
                    GL.glEnable(GL.GL_TEXTURE_2D)
                    GL.glBindTexture(GL.GL_TEXTURE_2D, texture)
                    GL.glColor4f(1, 1, 1, 1)
                    GL.glBegin(GL.GL_QUADS)
                    for u, v in ((0, 0), (1, 0), (1, 1), (0, 1)):
                        GL.glTexCoord2f(u, v)
                        GL.glVertex2f(u, v)
                    GL.glEnd()
                    GL.glBindTexture(GL.GL_TEXTURE_2D, 0)
                    GL.glDisable(GL.GL_TEXTURE_2D)

                glfw.swap_buffers(window)
                if not connected:
                    # Avoid spinning if the driver ignores vsync without a sender.
                    glfw.wait_events_timeout(0.02)
    finally:
        try:
            if texture:
                GL.glDeleteTextures([texture])
        finally:
            if window:
                glfw.destroy_window(window)
            glfw.terminate()


def main(argv=None):
    args = parse_args(argv)
    if sys.platform != "win32":
        print("Spout requires Windows.", file=sys.stderr)
        return 1
    try:
        run(args)
    except ImportError as exc:
        print(
            f"{exc}\nInstall dependencies with 64-bit Python 3.13:\n"
            "  py -3.13 -m pip install -r python/requirements-spout.txt",
            file=sys.stderr,
        )
        return 1
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        pass
    return 0


if __name__ == "__main__":
    sys.exit(main())
