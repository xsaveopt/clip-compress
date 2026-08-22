#!/usr/bin/env bash
set -euo pipefail

FFMPEG_VERSION=9.0.1
OPUS_VERSION=1.6.1
DAV1D_VERSION=1.5.4
NV_CODEC_VERSION=13.1.15.0

host=x86_64-w64-mingw32
root=$(cd "$(dirname "$0")" && pwd)
work=${WORK_DIR:-$root/.work}
out=${OUT_DIR:-$root/out}
prefix=$work/prefix

if [ -f "$out/ffmpeg.exe" ]; then
  echo "ffmpeg.exe already built"
  exit 0
fi

mkdir -p "$work" "$prefix" "$out"

export PKG_CONFIG_LIBDIR=$prefix/lib/pkgconfig
export PKG_CONFIG_PATH=$prefix/lib/pkgconfig

fetch() {
  local url=$1 dir=$2
  [ -d "$dir" ] && return 0
  mkdir -p "$dir"
  case "$url" in
    *.tar.xz) curl -fsSL "$url" | tar xJ -C "$dir" --strip-components=1 ;;
    *) curl -fsSL "$url" | tar xz -C "$dir" --strip-components=1 ;;
  esac
}

cat > "$work/mingw.cmake" <<EOF
set(CMAKE_SYSTEM_NAME Windows)
set(CMAKE_SYSTEM_PROCESSOR x86_64)
set(CMAKE_C_COMPILER $host-gcc)
set(CMAKE_CXX_COMPILER $host-g++)
set(CMAKE_RC_COMPILER $host-windres)
set(CMAKE_FIND_ROOT_PATH /usr/$host)
set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
EOF

cat > "$work/mingw.meson" <<EOF
[binaries]
c = '$host-gcc'
cpp = '$host-g++'
ar = '$host-ar'
strip = '$host-strip'
windres = '$host-windres'
pkg-config = 'pkg-config'

[host_machine]
system = 'windows'
cpu_family = 'x86_64'
cpu = 'x86_64'
endian = 'little'
EOF

fetch "https://github.com/xiph/opus/archive/refs/tags/v$OPUS_VERSION.tar.gz" "$work/opus"
cmake -S "$work/opus" -B "$work/opus-build" \
  -DCMAKE_TOOLCHAIN_FILE="$work/mingw.cmake" \
  -DCMAKE_INSTALL_PREFIX="$prefix" \
  -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF \
  -DOPUS_BUILD_PROGRAMS=OFF \
  -DOPUS_BUILD_TESTING=OFF
cmake --build "$work/opus-build" --parallel "$(nproc)"
cmake --install "$work/opus-build"

fetch "https://code.videolan.org/videolan/dav1d/-/archive/$DAV1D_VERSION/dav1d-$DAV1D_VERSION.tar.gz" "$work/dav1d"
meson setup "$work/dav1d-build" "$work/dav1d" \
  --cross-file "$work/mingw.meson" \
  --prefix "$prefix" \
  --buildtype release \
  --default-library static \
  -Denable_tools=false \
  -Denable_tests=false
ninja -C "$work/dav1d-build" install

fetch "https://github.com/FFmpeg/nv-codec-headers/archive/refs/tags/n$NV_CODEC_VERSION.tar.gz" "$work/nv-codec-headers"
make -C "$work/nv-codec-headers" PREFIX="$prefix" install

encoders=h264_nvenc,hevc_nvenc,av1_nvenc,aac,libopus,rawvideo,wrapped_avframe,pcm_s16le
decoders=h264,hevc,av1,libdav1d,vp8,vp9,wrapped_avframe,mpeg4,msmpeg4v3,wmv2,mjpeg,prores,ffv1,huffyuv,qtrle,rawvideo,aac,ac3,eac3,mp3,opus,libopus,vorbis,flac,alac,wmav2,pcm_s16le,pcm_s16be,pcm_s24le,pcm_u8,pcm_f32le
demuxers=mov,matroska,avi,lavfi
muxers=mp4,webm,null
parsers=h264,hevc,av1,vp8,vp9,aac,ac3,mpeg4video,opus,vorbis,flac,mjpeg
bsfs=extract_extradata,h264_mp4toannexb,hevc_mp4toannexb,aac_adtstoasc,vp9_superframe,av1_frame_split,av1_frame_merge,null
filters=abuffer,abuffersink,aformat,anull,aresample,asetpts,atrim,buffer,buffersink,copy,format,fps,null,scale,setpts,sine,testsrc,trim

fetch "https://ffmpeg.org/releases/ffmpeg-$FFMPEG_VERSION.tar.xz" "$work/ffmpeg"
mkdir -p "$work/ffmpeg-build"
(
  cd "$work/ffmpeg-build"
  "$work/ffmpeg/configure" \
    --prefix="$prefix" \
    --arch=x86_64 \
    --target-os=mingw32 \
    --cross-prefix="$host-" \
    --enable-cross-compile \
    --pkg-config=pkg-config \
    --pkg-config-flags=--static \
    --disable-autodetect \
    --disable-everything \
    --disable-doc \
    --disable-ffplay \
    --disable-ffprobe \
    --disable-network \
    --disable-debug \
    --disable-shared \
    --enable-static \
    --enable-zlib \
    --enable-avdevice \
    --enable-libopus \
    --enable-libdav1d \
    --enable-ffnvcodec \
    --enable-nvenc \
    --enable-encoder="$encoders" \
    --enable-decoder="$decoders" \
    --enable-demuxer="$demuxers" \
    --enable-muxer="$muxers" \
    --enable-parser="$parsers" \
    --enable-bsf="$bsfs" \
    --enable-filter="$filters" \
    --enable-indev=lavfi \
    --enable-protocol=file,pipe \
    --extra-ldflags="-static -static-libgcc"
  make -j "$(nproc)"
)

cp "$work/ffmpeg-build/ffmpeg.exe" "$out/ffmpeg.exe"
"$host-strip" "$out/ffmpeg.exe"

cp "$work/ffmpeg/COPYING.LGPLv2.1" "$out/ffmpeg-COPYING.LGPLv2.1.txt"
cat > "$out/ffmpeg-README.txt" <<EOF
ffmpeg.exe shipped with ClipCompress is an unmodified build of FFmpeg $FFMPEG_VERSION,
licensed under the GNU Lesser General Public License version 2.1 or later.
It is built without --enable-gpl and without --enable-nonfree.

Statically linked third-party libraries:
  libopus $OPUS_VERSION (BSD-3-Clause)   https://github.com/xiph/opus
  libdav1d $DAV1D_VERSION (BSD-2-Clause)   https://code.videolan.org/videolan/dav1d
  nv-codec-headers $NV_CODEC_VERSION (MIT)   https://github.com/FFmpeg/nv-codec-headers

Corresponding sources:
  https://ffmpeg.org/releases/ffmpeg-$FFMPEG_VERSION.tar.xz
  https://github.com/xiph/opus/archive/refs/tags/v$OPUS_VERSION.tar.gz
  https://code.videolan.org/videolan/dav1d/-/archive/$DAV1D_VERSION/dav1d-$DAV1D_VERSION.tar.gz
  https://github.com/FFmpeg/nv-codec-headers/archive/refs/tags/n$NV_CODEC_VERSION.tar.gz

The exact configure flags used are in build/ffmpeg/build.sh in the ClipCompress
source repository: https://github.com/xsaveopt/clip-compress
EOF

ls -l "$out"
