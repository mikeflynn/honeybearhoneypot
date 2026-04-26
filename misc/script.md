# Honey Bear Honey Pot: Video Walkthrough Outline

### 1. Introduction (0:00 - 1:00)

* The Hook: Start with a shot of the angry/glitching bear animation on the GUI alongside a terminal window showing "hacker" commands.

* What is it?: Briefly explain that this is `Honey Bear Honey Pot` — a fully functional SSH honeypot written in Go that simulates a Linux server to trap and monitor attackers.

* The Twist: Unlike most security tools which are purely command-line based or utilitarian, this one features a whimsical, animated GUI built with Fyne. It’s designed to run on desktop or even a physical "Hard Hat" with a screen.


### 2. Installation & Setup (1:00 - 2:30)

* Prerequisites: Briefly mention it needs Go (if building from source) or just a binary.

* Quick Install: Show the easiest installation method (Homebrew for Mac/Linux).
    ```bash
    brew tap mikeflynn/honeybearhoneypot
    brew install honeybearhoneypot
    ```

* First Run: Run the command `honeybearhoneypot` and show the GUI popping up immediately.

* The Config: Briefly show the help command (`-h`) to highlight key flags:
    *   Changing the port (`-ssh-port`).
    *   Headless mode for servers (`-no-gui`).
    *   Enabling/Disabling fun commands (`-no-fun`).


### 3. The "Attacker" Experience (2:30 - 5:00)

* Split Screen Demo: Set up a split-screen view.
    *   Left: The Honey Bear GUI (Admin view).
    *   Right: A standard terminal window (Attacker view).

* Login: SSH into the honeypot (`ssh root@localhost -p 1337`). Show that it accepts *any* password (the trap).

* Exploration: Run standard commands to show the simulation works:
    *   `ls`, `pwd`, `whoami`, `uname -a` `netstat` `sudo`
    *   Show the "fake" file system.
    *   Use env vars: export foo="bar"; echo "$foo"
    *   Classic hacker echo commands: echo -e "\x6F\x6B"


* The Bear Reactions:

    *   Execute a "suspicious" command (or just active typing) and highlight how the Bear's expression changes in the GUI (Sleeping -> Alert -> Typing -> Angry).

* Easter Eggs: Run the custom fun commands:
    *   `bearsay "Hello World"`
    *   Tab complete!
    *   `matrix` (Show the visual effect).
    *   `celebrate`.


### 4. The Admin Dashboard (5:00 - 6:30)

* Unlocking: Click the GUI to bring up the Keypad. Enter the PIN (Default/Configured).

* Stats Panel: Walk through the tabs:
    *   Live Stats: Current connections, total attacks.
    *   Logs: Viewing the history of commands run by attackers.
    *   App Control: Toggling fullscreen, resetting the session.

* Why this matters: Explain this gives you an at-a-glance view of your network's threat level without needing to tail log files.

* Deeper stats with export and analyze.

### 5. Advanced Features: CTF & Customization (6:30 - 8:00)

* Capture The Flag (CTF):
    *   Explain built-in CTF mode for educational events.
    *   Show an attacker running `ctf` to see challenges.
    *   Show `leaderboard` to see scores.

* Configuration: Briefly show `config.sample.json`.
    *   Mention you can customize the fake filesystem (add your own "secret" files for attackers to find).
    *   Customize CTF questions.

* Turning off silly functions.


### 6. Conclusion (8:00 - End)

* Use Cases:
    *   Education (learning SSH/Linux).
    *   Research (seeing what passwords bots are trying).
    *   Fun (putting it on a Raspberry Pi screen on your desk).

* Call to Action:
    *   "Check out the code on GitHub."
    *   "Star the repo."
    *   "Try installing it and let me know if you catch any bears."

----

  Production Notes & Assets
   * Visuals: Use the assets in internal/gui/assets/ (like bear_angry.jpg, bear_cool.jpg) for thumbnail art.     * Audio: Consider light, playful 8-bit or lo-fi background music to match the "whimsical" vibe, contrasting
     with the "hacking" subject matter.
     * Links to include:
       * Repo: github.com/mikeflynn/honeybearhoneypot
       * Site: honeybear.hydrox.fun

  Video Title Ideas
   * Honey Bear: The Cutest SSH Honeypot You'll Ever Meet
   * Catching Hackers with a Cartoon Bear (Honey Bear Walkthrough)
   * Building a "Hard Hat" Honeypot with Go and Fyne