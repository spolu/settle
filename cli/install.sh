#!/bin/sh

# If you are looking at this in your browser and would like to install Settle:
#
# MAC:
#   Open the terminal (look for the "Terminal" app) and type:
#     `curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle`
#
# LINUX:
#   Open a terminal and run:
#     `curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle`
#
# (This script is largely inspired by the Meteor install script whose license
#  is at https://github.com/meteor/meteor/blob/devel/LICENSE)

run_it () {

# This always does a clean install of the latest version of Settle into your
# ~/.settle, replacing whatever is already there.

RELEASE="0.0.2-pre"

## NOTE sh NOT bash. This script should be POSIX sh only, since we don't
## know what shell the user has. Debian uses 'dash' for 'sh', for
## example.

set -e
set -u

# Let's display everything on stderr.
exec 1>&2


# Check platform.
UNAME=$(uname)

if [ "$UNAME" != "Linux" -a "$UNAME" != "Darwin" ] ; then
    echo "Sorry, this OS is not supported yet via this installer."
    exit 1
fi

if [ "$UNAME" = "Darwin" ] ; then
  ### OSX ###
  if [ "i386" != "$(uname -p)" -o "1" != "$(sysctl -n hw.cpu64bit_capable 2>/dev/null || echo 0)" ] ; then
    # Can't just test uname -m = x86_64, because Snow Leopard can
    # return other values.
    echo "Only 64-bit Intel processors are supported at this time."
    exit 1
  fi
  PLATFORM="osx-x86_64"

elif [ "$UNAME" = "Linux" ] ; then
  ### Linux ###
  LINUX_ARCH=$(uname -m)
  if [ "${LINUX_ARCH}" = "x86_64" ] ; then
    PLATFORM="linux-x86_64"
  else
    echo "Unusable architecture: ${LINUX_ARCH}"
    echo "Only i686 and x86_64 are supported at this time."
    exit 1
  fi
fi

trap "echo Installation failed." EXIT

if [ -z $HOME ] || [ ! -d $HOME ]; then
  echo "The installation and use of Settle requires the \$HOME environment"
  echo "variable be set to a directory where its files can be installed."
  exit 1
fi

# If you already have a tropohouse/warehouse, we do a clean install here:
if [ -e "$HOME/.settle" ]; then
  echo "> Deleting your existing Settle executable."
  rm -rf "$HOME/.settle/settle"
else
  # Create the .settle directory
  echo "> Creating your local Settle directory $HOME/.settle."
  mkdir -p "$HOME/.settle"
fi

INSTALL_TMPDIR="$HOME/.settle/tmp"
BINARY_URL="https://github.com/spolu/settle/releases/download/${RELEASE}/settle.${RELEASE}.${PLATFORM}"
BINARY_FILE="$HOME/.settle/tmp/settle"

cleanUp() {
  rm -rf "$INSTALL_TMPDIR"
}

# Remove temporary files now in case they exist.
cleanUp

# Make sure cleanUp gets called if we exit abnormally.
trap cleanUp EXIT

mkdir -p "$INSTALL_TMPDIR"

# Only show progress bar animations if we have a tty
# (Prevents tons of console junk when installing within a pipe)
VERBOSITY="--silent";
if [ -t 1 ]; then
  VERBOSITY="--progress-bar"
fi

echo "> Downloading Settle binary."
# keep trying to curl the file until it works (resuming where possible)
MAX_ATTEMPTS=10
RETRY_DELAY_SECS=5
set +e
ATTEMPTS=0
while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]
do
  ATTEMPTS=$((ATTEMPTS + 1))

  curl -L $VERBOSITY --fail --continue-at - \
    "$BINARY_URL" --output "$BINARY_FILE"

  if [ $? -eq 0 ]
  then
      break
  fi

  echo "> Retrying download in $RETRY_DELAY_SECS seconds..."
  sleep $RETRY_DELAY_SECS
done
set -e

# bomb out if it didn't work, eg no net
test -e "${BINARY_FILE}"
chmod +x "${BINARY_FILE}"
mv "${BINARY_FILE}" "$HOME/.settle"

# just double-checking :)
test -x "$HOME/.settle/settle"

EXPORT_PATH="export PATH=\$PATH:$HOME/.settle"
if ["$PLATOFRM" = "linux-x86_64" ] ; then
  if ! grep -q "$EXPORT_PATH" "$HOME/.bashrc"; then
    echo "> Adding '$EXPORT_PATH' to your .bashrc file."
    touch "$HOME/.bashrc"
    echo "\n# settle path\n$EXPORT_PATH" >> "$HOME/.bashrc"
  fi
fi

if ["$PLATOFRM" = "osx-x86_64" ] ; then
  if ! grep -q "$EXPORT_PATH" "$HOME/.profile"; then
    echo "> Adding '$EXPORT_PATH' to your .profile file."
    touch "$HOME/.profile"
    echo "\n# settle path\n$EXPORT_PATH" >> "$HOME/.profile"
  fi
fi

# The `trap cleanUp EXIT` line above won't actually fire after the exec
# call below, so call cleanUp manually.
cleanUp

echo
echo "Settle ${RELEASE} has been installed locally (under ~/.settle)."
echo
echo "Read the command line help: 'settle help'"
echo "Register on a mint: 'settle register'"
echo

trap - EXIT
}

run_it
