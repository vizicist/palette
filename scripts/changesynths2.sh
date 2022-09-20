# for General MIDI 

find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"sound.synth\":\".*\",' '\"sound.synth\":\"005 Electric Piano 1\",'

find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"A-sound.synth\":\".*\",' '\"A-sound.synth\":\"005 Electric Piano 1\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"B-sound.synth\":\".*\",' '\"B-sound.synth\":\"001 Acoustic Grand Piano\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"C-sound.synth\":\".*\",' '\"C-sound.synth\":\"019 Rock Organ\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"D-sound.synth\":\".*\",' '\"D-sound.synth\":\"081 Lead 1 (square)\",'

# chall.sh '\"A-sound.synth\":\".*\",' '\"A-sound.synth\":\"0103 Ambient_E-Guitar\",' $@
# chall.sh '\"B-sound.synth\":\".*\",' '\"B-sound.synth\":\"0104 Dist_Bass 1\",' *.json
# chall.sh '\"C-sound.synth\":\".*\",' '\"C-sound.synth\":\"0105 Tacky 1\",' *.json
# chall.sh '\"D-sound.synth\":\".*\",' '\"D-sound.synth\":\"0101 Fantasy_Bell\",' *.json

# chall.sh '\"sound.synth\":\".*\",' '\"sound.synth\":\"0101 Fantasy_Bell\",' *.json
