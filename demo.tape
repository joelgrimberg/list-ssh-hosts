Output jumphost-demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 600
Set Theme "tokyonight"

# Show the SSH config file first
Type "cat ~/.ssh/config" Sleep 100ms Enter
Sleep 2s

# Show the program starting
Type "./jumphost" Sleep 100ms Enter
Sleep 2s

# Navigate through the host list
Down Sleep 400ms
Down Sleep 400ms
Down Sleep 400ms
Up Sleep 400ms
Sleep 1s

# Select a host
Enter
Sleep 2s

# Show password input screen
Type "mypassword123" Sleep 100ms
Sleep 1s
Enter
Sleep 2s

# Show spinner while connecting
Sleep 2s

# Show successful SSH connection
Type "whoami" Sleep 100ms Enter
Sleep 1s
Type "hostname" Sleep 100ms Enter
Sleep 1s
Type "pwd" Sleep 100ms Enter
Sleep 1s

# Exit SSH session
Type "exit" Sleep 100ms Enter
Sleep 1s 