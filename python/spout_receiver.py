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


PREVIEW_ACTIVE_EVENT = r"Local\Palette.SpoutPreviewActive"
PERFORMANCE_ACTIVE_EVENT = r"Local\Palette.SpoutPerformanceActive"


class PreviewGate:
    """Read a Palette display-state event without blocking the render loop."""

    def __init__(self, name=PREVIEW_ACTIVE_EVENT):
        import ctypes
        from ctypes import wintypes

        self._kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
        self._kernel32.CreateEventW.argtypes = [
            ctypes.c_void_p, wintypes.BOOL, wintypes.BOOL, wintypes.LPCWSTR
        ]
        self._kernel32.CreateEventW.restype = wintypes.HANDLE
        self._kernel32.WaitForSingleObject.argtypes = [wintypes.HANDLE, wintypes.DWORD]
        self._kernel32.WaitForSingleObject.restype = wintypes.DWORD
        self._kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
        self._kernel32.CloseHandle.restype = wintypes.BOOL
        self._handle = self._kernel32.CreateEventW(None, True, False, name)
        if not self._handle:
            raise RuntimeError("Could not open Palette Preview state")

    def active(self):
        result = self._kernel32.WaitForSingleObject(self._handle, 0)
        if result == 0:  # WAIT_OBJECT_0
            return True
        if result == 258:  # WAIT_TIMEOUT
            return False
        raise RuntimeError("Could not read Palette Preview state")

    def close(self):
        if self._handle:
            self._kernel32.CloseHandle(self._handle)
            self._handle = None


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
    parser.add_argument(
        "--preview-only", action="store_true",
        help="show black unless a Palette pad is in Preview mode",
    )
    parser.add_argument(
        "--performance-only", action="store_true",
        help="show the window only while the Palette GUI is in Performance mode",
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
    preview_gate = None
    performance_gate = None
    try:
        if args.preview_only:
            preview_gate = PreviewGate()
        if args.performance_only:
            performance_gate = PreviewGate(PERFORMANCE_ACTIVE_EVENT)
        # OpenGL 2.1 provides the compatibility functions used to draw the quad.
        glfw.window_hint(glfw.CONTEXT_VERSION_MAJOR, 2)
        glfw.window_hint(glfw.CONTEXT_VERSION_MINOR, 1)
        glfw.window_hint(glfw.DECORATED, glfw.FALSE)
        glfw.window_hint(
            glfw.FLOATING, glfw.TRUE if args.always_on_top else glfw.FALSE
        )
        # Position the window before showing it to avoid an initial flash elsewhere.
        glfw.window_hint(glfw.VISIBLE, glfw.FALSE)
        if args.performance_only:
            glfw.window_hint(glfw.FOCUS_ON_SHOW, glfw.FALSE)
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
        visible = performance_gate is None or performance_gate.active()
        if visible:
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
                should_show = performance_gate is None or performance_gate.active()
                if should_show != visible:
                    if should_show:
                        glfw.show_window(window)
                    else:
                        glfw.hide_window(window)
                    visible = should_show
                if not visible:
                    glfw.wait_events_timeout(0.05)
                    continue
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
                if received and connected and (preview_gate is None or preview_gate.active()):
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
            if preview_gate is not None:
                preview_gate.close()
            if performance_gate is not None:
                performance_gate.close()


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
