#!/bin/sh
# Creates a new major, minor or patch release and creates tarballs for each
# architecture.
set -e

major="$(cut -d. -f1 version.txt)"
minor="$(cut -d. -f2 version.txt)"
patch="$(cut -d. -f3 version.txt)"
version="$major.$minor.$patch"

case "$1" in
"major")
    new_major="$((major+1))"
    new_version="$new_major.$minor.$patch"
    ;;
"minor")
    new_minor="$((minor+1))"
    new_version="$major.$new_minor.$patch"
    ;;
"patch")
    new_patch="$((patch+1))"
    new_version="$major.$minor.$new_patch"
    ;;
*)
    echo "major, minor or patch not specified" > /dev/stderr
    exit 2
    ;;
esac

echo "Updating $version -> $new_version"
echo "$new_version" > version.txt

# We cross-compile binaries to all in arch.txt, use those binaries for each
# arch-specific release rather than recompiling twice
echo "Making binaries..."
tmp_prefix="$(mktemp -d)"
make install PREFIX="$tmp_prefix"

for arch in $(grep -v '^#' arch.txt | awk 'NF = 2 { printf "%s-%s\n", $1, $2 }' | sort | uniq); do
    echo "Creating release directory for $arch..."
    release_dir="./call-buddy-$new_version/"
    mkdir "$release_dir"

    printf "creating structure... "
    # Copy reference directory into arch-specfic release dir
    cp -RL "$tmp_prefix/" "$release_dir/"

    printf "replacing call-buddy with arch-specific binary... "
    # Remove our machine's arch-specific binary
    rm "$release_dir/bin/call-buddy"
    # Replace with correct arch-specific binary
    cp "$tmp_prefix/lib/call-buddy/$arch/call-buddy" "$release_dir/bin/"

    if [ "$arch" = "windows-amd64" ] || [ "$arch" = "windows-386" ]; then
        # Windows support is pretty terrible, but we'll try
        mv "$release_dir/bin/call-buddy" "$release_dir/bin/call-buddy.exe"
    fi

    printf "creating tarball... "
    # Create a tarball
    tar -czf "call-buddy-$new_version.$arch".tar.gz "$release_dir/"
    rm -r "$release_dir"

    echo "done"
done

rm -r "$tmp_prefix"
