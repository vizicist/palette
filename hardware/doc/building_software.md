<h1>Building the Software of the Space Palette Pro</h1>
These are the steps to install and configure all the necessary software for a full-blown (visuals and sound) Space Palette Pro.  This procedure is much more complex than it is for a visuals-only Space Palette Pro.  If you only want to do visuals, the much simpler installation procedure is at
<a href="https://github.com/vizicist/palette/blob/main/doc/using_resolume.md">https://github.com/vizicist/palette/blob/main/doc/using_resolume.md</a>
<p>

<ol>
<li>Install LoopBe30 from <a href="https://www.nerds.de/en/order.html">https://www.nerds.de/en/order.html</a><br>
    <ul>
    <li>Use LoopBe30 tray item to expand "ports after reboot" to 16
    <li>Turn off "Enable Shortcut Detection"
    <li>Reboot
    </ul>
<li>Install ASIO4ALL from <a href="https://asio4all.org">https://asio4all.org</a>.
<li>Install Palette
    <ul>
    <li>Use the desired (usually the latest) install package in
    <a href="https://github.com/vizicist/palette/tree/main/release">https://github.com/vizicist/palette/tree/main/release</a>
    </ul>
<li>Create (if it doesn't exist already) this
directory: c:\users\public\documents\vstplugins64.
In the Windows Explorer, it should appear as
C: > Users > Public > Public Documents > vstplugins64
<li>Install Omnisphere 2 in c:\users\public\documents\vstplugins64
<li>Install Battery 4 in c:\users\public\documents\vstplugins64
<li>Install Battery 4 Factory Library
<li>Install Plogue Bidule:
    <ul>
    <li>In Edit->Preferences->VST set the VST plugin path to c:\users\public\documents\vstplugins64
    <li>In Tools->Osc Server, enable OSC 
    <li>Open %LOCALAPPDATA%\Palette\config\palette.bidule
    <li>Make sure Battery 4 instances can send audio, fixing the audio device
if necessary to make sure there is minimal latency when playing back drum hits.
This may require using the Bidule GUI to set the audio output device to a suitably-configured ASIO device.
    <li>Authorize Omnisphere 2 by double-clicking on one of its instances in Bidule, and following the instructions.  The window with the authorization instructions may be hidden behind other windows.
    <li>Restart Bidule, verify that Omnisphere is authorized by double-clicking on one of the Omnisphere instances and verifying that it no longer shows the auuthorization screen.
    </ul>
<li>Install Resolume 7 (Avenue)  
    <ul>
    <li>In Avenue->Preferences->Video, add C:\Program Files\Palette\ffgl
    <li>In Avenue->Preferences->OSC, enable OSC Input on port 7000
    <li>Quit and restart Resolume
    <li>Verify that Palette plugin is seen in Sources under Generators
    <li>In Composition->Open, open %LOCALAPPDATA%\Palette\config\PaletteABCD.avc, then Quit
    <li>In Output->Fullscreen, set the Palette's main monitor to fullscreen
    </ul>
<li>Install SenselApp
    <ul>
    <li>Verify that your Morphs are seen, and updated to the latest firmware.
    <li>If you are using 4 Morphs (e.g. a full-blown Space Palette Pro controller), make sure their serial numbers are in %LOCALAPPDATA%\Palette\config\morphs.json and are associated with the desired pad letters (A,B,C,D)
    </ul>
<li>Install Git from <a href="https://gitforwindows.org">https://gitforwindows.org</a><br>
    <ul>
    <li>Accept all the defaults
    <li>Add c:\program files\git\usr\bin to the value of PATH in System Properties->Environment Variables
    <li>Optional: install GitHub Desktop from <a href="https://desktop.github.com">https://desktop.github.com</a>
    </ul>
</ol>
If you are using a touchscreen monitor for the preset selector GUI, then do these steps:
<p>
<ol>
<li>Use Windows Settings->Display to adjust the monitor settings
    <ul>
    <li>Adjust touchscreen monitor for Resolution = 800x1280, Orientation = Portrait (flipped)
    <li>Put console monitor on the right, touchscreen monitor in the middle, and the external monitor (or oval monitor) on the left.
    </ul>
<li>Make sure Windows knows which monitor is the touchscreen
    <ul>
    <li>Go to the Windows Control Panel (the old one)
    <li>change the "View by" to "Small icons"
    <li>In "Tablet PC Settings", select "Setup..." and then "Touch Input",
      following the instructions to select the touchscreen monitor as the touch screen
    <li> If you don't see "Tablet PC Settings", it means the touchscreen monitor
      isn't connected.
    </ul>
</ol>

You should now be able to use the Palette as described in
<a href="https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md">https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md</a>