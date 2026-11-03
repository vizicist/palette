import ctypes
import contextlib
import io
from ctypes import wintypes
import sys
import unittest
import uuid
from unittest import mock

from spout_receiver import PreviewGate, parse_args, run


@unittest.skipUnless(sys.platform == "win32", "Spout state uses Windows events")
class PreviewGateTests(unittest.TestCase):
    def test_default_blank_shared_state_and_receiver_restarts(self):
        name = "Local\\Palette.SpoutPreviewTest." + uuid.uuid4().hex
        kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
        kernel32.SetEvent.argtypes = [wintypes.HANDLE]
        kernel32.SetEvent.restype = wintypes.BOOL
        kernel32.ResetEvent.argtypes = [wintypes.HANDLE]
        kernel32.ResetEvent.restype = wintypes.BOOL
        first = PreviewGate(name)
        self.addCleanup(first.close)
        self.assertFalse(first.active())
        second = PreviewGate(name)
        self.addCleanup(second.close)
        self.assertFalse(second.active())
        self.assertTrue(kernel32.SetEvent(first._handle))
        self.assertTrue(first.active())
        self.assertTrue(second.active())
        second.close()
        restarted = PreviewGate(name)
        self.addCleanup(restarted.close)
        self.assertTrue(restarted.active())
        self.assertTrue(kernel32.ResetEvent(first._handle))
        self.assertFalse(first.active())
        self.assertFalse(restarted.active())

    def test_preview_gate_is_opt_in(self):
        self.assertFalse(parse_args(["--sender", "Avenue - Screen 3"]).preview_only)
        self.assertTrue(parse_args([
            "--sender", "Avenue - Screen 2", "--preview-only"
        ]).preview_only)

    def test_performance_mode_visibility_is_opt_in_for_both_displays(self):
        for sender in ["Avenue - Screen 2", "Avenue - Screen 3"]:
            self.assertFalse(parse_args(["--sender", sender]).performance_only)
            self.assertTrue(parse_args([
                "--sender", sender, "--performance-only"
            ]).performance_only)

    def test_receiver_hides_outside_performance_without_stopping_the_window(self):
        # Exercise the real receiver loop with graphics/Spout doubles so this
        # check never opens or operates a window on the user's desktop.
        glfw, spout, opengl = mock.MagicMock(), mock.MagicMock(), mock.MagicMock()
        gl, gate = mock.MagicMock(), mock.MagicMock()
        opengl.GL = gl
        gl.glGenTextures.return_value = 1
        glfw.init.return_value = True
        glfw.get_framebuffer_size.return_value = (384, 216)
        frame = {"index": 0}

        def poll():
            frame["index"] += 1

        glfw.poll_events.side_effect = poll
        glfw.window_should_close.side_effect = lambda _: frame["index"] >= 4
        gate.active.side_effect = lambda: frame["index"] == 2
        receiver = spout.SpoutReceiver.return_value
        receiver.__enter__.return_value = receiver
        receiver.isUpdated.return_value = False
        receiver.receiveTexture.return_value = True
        receiver.isConnected.return_value = True
        with mock.patch.dict(sys.modules, {
            "glfw": glfw, "SpoutGL": spout, "OpenGL": opengl
        }), mock.patch("spout_receiver.PreviewGate", return_value=gate), contextlib.redirect_stdout(io.StringIO()):
            run(parse_args(["--sender", "Test", "--performance-only"]))
        glfw.create_window.assert_called_once()
        glfw.show_window.assert_called_once()
        glfw.hide_window.assert_called_once()
        receiver.receiveTexture.assert_called_once()
        gl.glBegin.assert_called_once()
        gate.close.assert_called_once()


if __name__ == "__main__":
    unittest.main()
