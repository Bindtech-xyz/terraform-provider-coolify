#!/usr/bin/env bash
# Résout le fichier de secrets à manipuler.
#
# Règle : exiger un choix seulement quand il y a un choix à faire.
#   - FILE fourni          → ce fichier, sans discussion
#   - un seul fichier      → celui-là, inutile de le nommer
#   - plusieurs fichiers   → refuser, et lister les possibilités
#
# Un défaut arbitraire ferait éditer le mauvais fichier en silence ; exiger un
# paramètre quand il n'y a qu'un candidat n'apporte rien. Ni l'un ni l'autre.
set -euo pipefail

dir="${1:?répertoire des secrets attendu}"
want="${2:-}"

if [ -n "$want" ]; then
    printf '%s/%s.enc.yaml\n' "$dir" "$want"
    exit 0
fi

shopt -s nullglob
files=("$dir"/*.enc.yaml)

case ${#files[@]} in
0)
    echo "Aucun fichier de secrets dans $dir." >&2
    echo "En créer un : task secrets:create FILE=<domaine>" >&2
    exit 1
    ;;
1)
    printf '%s\n' "${files[0]}"
    ;;
*)
    echo "Plusieurs fichiers de secrets — préciser lequel :" >&2
    for f in "${files[@]}"; do
        name=$(basename "$f" .enc.yaml)
        echo "    FILE=$name" >&2
    done
    exit 1
    ;;
esac
