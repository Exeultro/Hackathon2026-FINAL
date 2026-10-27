#!/usr/bin/env bash
# Generates silent slide videos for the mock: scripts/demo-video.sh
set -euo pipefail
cd "$(dirname "$0")/.."
FONT=/usr/share/fonts/noto/NotoSans-Regular.ttf
BOLD=/usr/share/fonts/noto/NotoSans-Bold.ttf

make() { # out duration title "start|end|slide" ...
  local out=$1 dur=$2 title=$3; shift 3
  local f="drawtext=fontfile=$FONT:text='$title':fontcolor=0x9aa4b2:fontsize=20:x=32:y=28"
  for s in "$@"; do
    IFS='|' read -r a b text <<<"$s"
    f+=",drawtext=fontfile=$BOLD:text='$text':fontcolor=white:fontsize=40:x=(w-tw)/2:y=(h-th)/2:enable='between(t,$a,$b)'"
  done
  f+=",drawtext=fontfile=$FONT:text='%{pts\:hms}':fontcolor=0x6b7585:fontsize=18:x=w-tw-32:y=h-46"
  ffmpeg -hide_banner -loglevel error -y -f lavfi -i "color=c=0x1b2230:s=960x540:r=10:d=$dur" \
    -vf "$f" -c:v libx264 -preset veryfast -crf 32 -pix_fmt yuv420p -movflags +faststart "public/demo/$out"
}

make pythagoras.mp4 180 'Геометрия, лекция 3' \
  '0|40|Прямоугольный треугольник' '40|56|c² = a² + b²' '56|72|3² + 4² = 5²' '72|100|Пифагоровы тройки' \
  '100|150|Доказательство через площади' '150|180|Применение'
make derivative.mp4 150 'Алгебра, лекция 7' \
  '0|45|Скорость изменения' '45|81|f′(x) = lim Δy/Δx' '81|95|(x²)′ = 2x' '95|150|Правила дифференцирования'

for v in pythagoras:62 derivative:70; do
  ffmpeg -hide_banner -loglevel error -y -ss "${v##*:}" -i "public/demo/${v%%:*}.mp4" -frames:v 1 -vf scale=640:-1 -q:v 4 "public/demo/${v%%:*}.jpg"
done
