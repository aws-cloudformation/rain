#compdef rain

_arguments \
  '1: :->level1' \
  '2: :_files'
case $state in
  level1)
    case $words[1] in
      rain)
        _arguments '1: :(cat check deploy diff fmt help logs ls rm tree version watch)'
      ;;
      *)
        _arguments '*: :_files'
      ;;
    esac
  ;;
  *)
    _arguments '*: :_files'
  ;;
esac
