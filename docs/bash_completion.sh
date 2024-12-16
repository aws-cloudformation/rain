# bash completion for rain                                 -*- shell-script -*-

__rain_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE:-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Homebrew on Macs have version 1.3 of bash-completion which doesn't include
# _init_completion. This is a very minimal version of that function.
__rain_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

__rain_index_of_word()
{
    local w word=$1
    shift
    index=0
    for w in "$@"; do
        [[ $w = "$word" ]] && return
        index=$((index+1))
    done
    index=-1
}

__rain_contains_word()
{
    local w word=$1; shift
    for w in "$@"; do
        [[ $w = "$word" ]] && return
    done
    return 1
}

__rain_handle_go_custom_completion()
{
    __rain_debug "${FUNCNAME[0]}: cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}"

    local shellCompDirectiveError=1
    local shellCompDirectiveNoSpace=2
    local shellCompDirectiveNoFileComp=4
    local shellCompDirectiveFilterFileExt=8
    local shellCompDirectiveFilterDirs=16

    local out requestComp lastParam lastChar comp directive args

    # Prepare the command to request completions for the program.
    # Calling ${words[0]} instead of directly rain allows handling aliases
    args=("${words[@]:1}")
    # Disable ActiveHelp which is not supported for bash completion v1
    requestComp="RAIN_ACTIVE_HELP=0 ${words[0]} __completeNoDesc ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __rain_debug "${FUNCNAME[0]}: lastParam ${lastParam}, lastChar ${lastChar}"

    if [ -z "${cur}" ] && [ "${lastChar}" != "=" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go method.
        __rain_debug "${FUNCNAME[0]}: Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __rain_debug "${FUNCNAME[0]}: calling ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval "${requestComp}" 2>/dev/null)

    # Extract the directive integer at the very end of the output following a colon (:)
    directive=${out##*:}
    # Remove the directive
    out=${out%:*}
    if [ "${directive}" = "${out}" ]; then
        # There is not directive specified
        directive=0
    fi
    __rain_debug "${FUNCNAME[0]}: the completion directive is: ${directive}"
    __rain_debug "${FUNCNAME[0]}: the completions are: ${out}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        # Error code.  No completion.
        __rain_debug "${FUNCNAME[0]}: received error from custom completion go code"
        return
    else
        if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __rain_debug "${FUNCNAME[0]}: activating no space"
                compopt -o nospace
            fi
        fi
        if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __rain_debug "${FUNCNAME[0]}: activating no file completion"
                compopt +o default
            fi
        fi
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local fullFilter filter filteringCmd
        # Do not use quotes around the $out variable or else newline
        # characters will be kept.
        for filter in ${out}; do
            fullFilter+="$filter|"
        done

        filteringCmd="_filedir $fullFilter"
        __rain_debug "File filtering command: $filteringCmd"
        $filteringCmd
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        # Use printf to strip any trailing newline
        subdir=$(printf "%s" "${out}")
        if [ -n "$subdir" ]; then
            __rain_debug "Listing directories in $subdir"
            __rain_handle_subdirs_in_dir_flag "$subdir"
        else
            __rain_debug "Listing directories in ."
            _filedir -d
        fi
    else
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${out}" -- "$cur")
    fi
}

__rain_handle_reply()
{
    __rain_debug "${FUNCNAME[0]}"
    local comp
    case $cur in
        -*)
            if [[ $(type -t compopt) = "builtin" ]]; then
                compopt -o nospace
            fi
            local allflags
            if [ ${#must_have_one_flag[@]} -ne 0 ]; then
                allflags=("${must_have_one_flag[@]}")
            else
                allflags=("${flags[*]} ${two_word_flags[*]}")
            fi
            while IFS='' read -r comp; do
                COMPREPLY+=("$comp")
            done < <(compgen -W "${allflags[*]}" -- "$cur")
            if [[ $(type -t compopt) = "builtin" ]]; then
                [[ "${COMPREPLY[0]}" == *= ]] || compopt +o nospace
            fi

            # complete after --flag=abc
            if [[ $cur == *=* ]]; then
                if [[ $(type -t compopt) = "builtin" ]]; then
                    compopt +o nospace
                fi

                local index flag
                flag="${cur%=*}"
                __rain_index_of_word "${flag}" "${flags_with_completion[@]}"
                COMPREPLY=()
                if [[ ${index} -ge 0 ]]; then
                    PREFIX=""
                    cur="${cur#*=}"
                    ${flags_completion[${index}]}
                    if [ -n "${ZSH_VERSION:-}" ]; then
                        # zsh completion needs --flag= prefix
                        eval "COMPREPLY=( \"\${COMPREPLY[@]/#/${flag}=}\" )"
                    fi
                fi
            fi

            if [[ -z "${flag_parsing_disabled}" ]]; then
                # If flag parsing is enabled, we have completed the flags and can return.
                # If flag parsing is disabled, we may not know all (or any) of the flags, so we fallthrough
                # to possibly call handle_go_custom_completion.
                return 0;
            fi
            ;;
    esac

    # check if we are handling a flag with special work handling
    local index
    __rain_index_of_word "${prev}" "${flags_with_completion[@]}"
    if [[ ${index} -ge 0 ]]; then
        ${flags_completion[${index}]}
        return
    fi

    # we are parsing a flag and don't have a special handler, no completion
    if [[ ${cur} != "${words[cword]}" ]]; then
        return
    fi

    local completions
    completions=("${commands[@]}")
    if [[ ${#must_have_one_noun[@]} -ne 0 ]]; then
        completions+=("${must_have_one_noun[@]}")
    elif [[ -n "${has_completion_function}" ]]; then
        # if a go completion function is provided, defer to that function
        __rain_handle_go_custom_completion
    fi
    if [[ ${#must_have_one_flag[@]} -ne 0 ]]; then
        completions+=("${must_have_one_flag[@]}")
    fi
    while IFS='' read -r comp; do
        COMPREPLY+=("$comp")
    done < <(compgen -W "${completions[*]}" -- "$cur")

    if [[ ${#COMPREPLY[@]} -eq 0 && ${#noun_aliases[@]} -gt 0 && ${#must_have_one_noun[@]} -ne 0 ]]; then
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${noun_aliases[*]}" -- "$cur")
    fi

    if [[ ${#COMPREPLY[@]} -eq 0 ]]; then
        if declare -F __rain_custom_func >/dev/null; then
            # try command name qualified custom func
            __rain_custom_func
        else
            # otherwise fall back to unqualified for compatibility
            declare -F __custom_func >/dev/null && __custom_func
        fi
    fi

    # available in bash-completion >= 2, not always present on macOS
    if declare -F __ltrim_colon_completions >/dev/null; then
        __ltrim_colon_completions "$cur"
    fi

    # If there is only 1 completion and it is a flag with an = it will be completed
    # but we don't want a space after the =
    if [[ "${#COMPREPLY[@]}" -eq "1" ]] && [[ $(type -t compopt) = "builtin" ]] && [[ "${COMPREPLY[0]}" == --*= ]]; then
       compopt -o nospace
    fi
}

# The arguments should be in the form "ext1|ext2|extn"
__rain_handle_filename_extension_flag()
{
    local ext="$1"
    _filedir "@(${ext})"
}

__rain_handle_subdirs_in_dir_flag()
{
    local dir="$1"
    pushd "${dir}" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
}

__rain_handle_flag()
{
    __rain_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    # if a command required a flag, and we found it, unset must_have_one_flag()
    local flagname=${words[c]}
    local flagvalue=""
    # if the word contained an =
    if [[ ${words[c]} == *"="* ]]; then
        flagvalue=${flagname#*=} # take in as flagvalue after the =
        flagname=${flagname%=*} # strip everything after the =
        flagname="${flagname}=" # but put the = back
    fi
    __rain_debug "${FUNCNAME[0]}: looking for ${flagname}"
    if __rain_contains_word "${flagname}" "${must_have_one_flag[@]}"; then
        must_have_one_flag=()
    fi

    # if you set a flag which only applies to this command, don't show subcommands
    if __rain_contains_word "${flagname}" "${local_nonpersistent_flags[@]}"; then
      commands=()
    fi

    # keep flag value with flagname as flaghash
    # flaghash variable is an associative array which is only supported in bash > 3.
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        if [ -n "${flagvalue}" ] ; then
            flaghash[${flagname}]=${flagvalue}
        elif [ -n "${words[ $((c+1)) ]}" ] ; then
            flaghash[${flagname}]=${words[ $((c+1)) ]}
        else
            flaghash[${flagname}]="true" # pad "true" for bool flag
        fi
    fi

    # skip the argument to a two word flag
    if [[ ${words[c]} != *"="* ]] && __rain_contains_word "${words[c]}" "${two_word_flags[@]}"; then
        __rain_debug "${FUNCNAME[0]}: found a flag ${words[c]}, skip the next argument"
        c=$((c+1))
        # if we are looking for a flags value, don't show commands
        if [[ $c -eq $cword ]]; then
            commands=()
        fi
    fi

    c=$((c+1))

}

__rain_handle_noun()
{
    __rain_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    if __rain_contains_word "${words[c]}" "${must_have_one_noun[@]}"; then
        must_have_one_noun=()
    elif __rain_contains_word "${words[c]}" "${noun_aliases[@]}"; then
        must_have_one_noun=()
    fi

    nouns+=("${words[c]}")
    c=$((c+1))
}

__rain_handle_command()
{
    __rain_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    local next_command
    if [[ -n ${last_command} ]]; then
        next_command="_${last_command}_${words[c]//:/__}"
    else
        if [[ $c -eq 0 ]]; then
            next_command="_rain_root_command"
        else
            next_command="_${words[c]//:/__}"
        fi
    fi
    c=$((c+1))
    __rain_debug "${FUNCNAME[0]}: looking for ${next_command}"
    declare -F "$next_command" >/dev/null && $next_command
}

__rain_handle_word()
{
    if [[ $c -ge $cword ]]; then
        __rain_handle_reply
        return
    fi
    __rain_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"
    if [[ "${words[c]}" == -* ]]; then
        __rain_handle_flag
    elif __rain_contains_word "${words[c]}" "${commands[@]}"; then
        __rain_handle_command
    elif [[ $c -eq 0 ]]; then
        __rain_handle_command
    elif __rain_contains_word "${words[c]}" "${command_aliases[@]}"; then
        # aliashash variable is an associative array which is only supported in bash > 3.
        if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
            words[c]=${aliashash[${words[c]}]}
            __rain_handle_command
        else
            __rain_handle_noun
        fi
    else
        __rain_handle_noun
    fi
    __rain_handle_word
}

_rain_bootstrap()
{
    last_command="rain_bootstrap"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-owner=")
    two_word_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_build()
{
    last_command="rain_build"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--bare")
    flags+=("-b")
    local_nonpersistent_flags+=("--bare")
    local_nonpersistent_flags+=("-b")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--json")
    flags+=("-j")
    local_nonpersistent_flags+=("--json")
    local_nonpersistent_flags+=("-j")
    flags+=("--list")
    flags+=("-l")
    local_nonpersistent_flags+=("--list")
    local_nonpersistent_flags+=("-l")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--no-cache")
    local_nonpersistent_flags+=("--no-cache")
    flags+=("--omit-patches")
    local_nonpersistent_flags+=("--omit-patches")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--pkl-class")
    local_nonpersistent_flags+=("--pkl-class")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--prompt")
    local_nonpersistent_flags+=("--prompt")
    flags+=("--prompt-lang=")
    two_word_flags+=("--prompt-lang")
    local_nonpersistent_flags+=("--prompt-lang")
    local_nonpersistent_flags+=("--prompt-lang=")
    flags+=("--recommend")
    local_nonpersistent_flags+=("--recommend")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--schema")
    flags+=("-s")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("-s")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cat()
{
    last_command="rain_cat"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config")
    flags+=("-c")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("-c")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--transformed")
    flags+=("-t")
    local_nonpersistent_flags+=("--transformed")
    local_nonpersistent_flags+=("-t")
    flags+=("--unformatted")
    flags+=("-u")
    local_nonpersistent_flags+=("--unformatted")
    local_nonpersistent_flags+=("-u")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cc_deploy()
{
    last_command="rain_cc_deploy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    local_nonpersistent_flags+=("-c")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--ignore-unknown-params")
    local_nonpersistent_flags+=("--ignore-unknown-params")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--unlock=")
    two_word_flags+=("--unlock")
    two_word_flags+=("-u")
    local_nonpersistent_flags+=("--unlock")
    local_nonpersistent_flags+=("--unlock=")
    local_nonpersistent_flags+=("-u")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cc_drift()
{
    last_command="rain_cc_drift"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cc_help()
{
    last_command="rain_cc_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_rain_cc_rm()
{
    last_command="rain_cc_rm"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cc_state()
{
    last_command="rain_cc_state"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_cc()
{
    last_command="rain_cc"

    command_aliases=()

    commands=()
    commands+=("deploy")
    commands+=("drift")
    commands+=("help")
    commands+=("rm")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ccdel")
        aliashash["ccdel"]="rm"
        command_aliases+=("ccdelete")
        aliashash["ccdelete"]="rm"
        command_aliases+=("ccremove")
        aliashash["ccremove"]="rm"
    fi
    commands+=("state")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-owner=")
    two_word_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_console()
{
    last_command="rain_console"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--logout")
    flags+=("-l")
    local_nonpersistent_flags+=("--logout")
    local_nonpersistent_flags+=("-l")
    flags+=("--name=")
    two_word_flags+=("--name")
    two_word_flags+=("-n")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    local_nonpersistent_flags+=("-n")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--service=")
    two_word_flags+=("--service")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--service")
    local_nonpersistent_flags+=("--service=")
    local_nonpersistent_flags+=("-s")
    flags+=("--url")
    flags+=("-u")
    local_nonpersistent_flags+=("--url")
    local_nonpersistent_flags+=("-u")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_deploy()
{
    last_command="rain_deploy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--changeset")
    local_nonpersistent_flags+=("--changeset")
    flags+=("--config=")
    two_word_flags+=("--config")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    local_nonpersistent_flags+=("-c")
    flags+=("--detach")
    flags+=("-d")
    local_nonpersistent_flags+=("--detach")
    local_nonpersistent_flags+=("-d")
    flags+=("--experimental")
    local_nonpersistent_flags+=("--experimental")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--ignore-unknown-params")
    local_nonpersistent_flags+=("--ignore-unknown-params")
    flags+=("--keep")
    flags+=("-k")
    local_nonpersistent_flags+=("--keep")
    local_nonpersistent_flags+=("-k")
    flags+=("--nested-change-set")
    local_nonpersistent_flags+=("--nested-change-set")
    flags+=("--no-exec")
    flags+=("-x")
    local_nonpersistent_flags+=("--no-exec")
    local_nonpersistent_flags+=("-x")
    flags+=("--node-style=")
    two_word_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style=")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--role-arn=")
    two_word_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn=")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-owner=")
    two_word_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--termination-protection")
    flags+=("-t")
    local_nonpersistent_flags+=("--termination-protection")
    local_nonpersistent_flags+=("-t")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_diff()
{
    last_command="rain_diff"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--long")
    flags+=("-l")
    local_nonpersistent_flags+=("--long")
    local_nonpersistent_flags+=("-l")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_fmt()
{
    last_command="rain_fmt"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--datamodel")
    local_nonpersistent_flags+=("--datamodel")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--json")
    flags+=("-j")
    local_nonpersistent_flags+=("--json")
    local_nonpersistent_flags+=("-j")
    flags+=("--node-style=")
    two_word_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style=")
    flags+=("--pkl")
    flags+=("-p")
    local_nonpersistent_flags+=("--pkl")
    local_nonpersistent_flags+=("-p")
    flags+=("--pkl-basic")
    local_nonpersistent_flags+=("--pkl-basic")
    flags+=("--pkl-package=")
    two_word_flags+=("--pkl-package")
    local_nonpersistent_flags+=("--pkl-package")
    local_nonpersistent_flags+=("--pkl-package=")
    flags+=("--unsorted")
    flags+=("-u")
    local_nonpersistent_flags+=("--unsorted")
    local_nonpersistent_flags+=("-u")
    flags+=("--verify")
    flags+=("-v")
    local_nonpersistent_flags+=("--verify")
    local_nonpersistent_flags+=("-v")
    flags+=("--write")
    flags+=("-w")
    local_nonpersistent_flags+=("--write")
    local_nonpersistent_flags+=("-w")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_forecast()
{
    last_command="rain_forecast"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--action=")
    two_word_flags+=("--action")
    local_nonpersistent_flags+=("--action")
    local_nonpersistent_flags+=("--action=")
    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--config=")
    two_word_flags+=("--config")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    local_nonpersistent_flags+=("-c")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--ignore=")
    two_word_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore=")
    flags+=("--include-iam")
    local_nonpersistent_flags+=("--include-iam")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--plugin=")
    two_word_flags+=("--plugin")
    local_nonpersistent_flags+=("--plugin")
    local_nonpersistent_flags+=("--plugin=")
    flags+=("--plugin-only")
    local_nonpersistent_flags+=("--plugin-only")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--role-arn=")
    two_word_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_help()
{
    last_command="rain_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_rain_info()
{
    last_command="rain_info"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--creds")
    flags+=("-c")
    local_nonpersistent_flags+=("--creds")
    local_nonpersistent_flags+=("-c")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_logs()
{
    last_command="rain_logs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--chart")
    flags+=("-c")
    local_nonpersistent_flags+=("--chart")
    local_nonpersistent_flags+=("-c")
    flags+=("--days=")
    two_word_flags+=("--days")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--days")
    local_nonpersistent_flags+=("--days=")
    local_nonpersistent_flags+=("-d")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--length=")
    two_word_flags+=("--length")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--length")
    local_nonpersistent_flags+=("--length=")
    local_nonpersistent_flags+=("-l")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--since-user-initiated")
    flags+=("-s")
    local_nonpersistent_flags+=("--since-user-initiated")
    local_nonpersistent_flags+=("-s")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_ls()
{
    last_command="rain_ls"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--changeset")
    flags+=("-c")
    local_nonpersistent_flags+=("--changeset")
    local_nonpersistent_flags+=("-c")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_merge()
{
    last_command="rain_merge"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--node-style=")
    two_word_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style=")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_module_bootstrap()
{
    last_command="rain_module_bootstrap"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--domain=")
    two_word_flags+=("--domain")
    local_nonpersistent_flags+=("--domain")
    local_nonpersistent_flags+=("--domain=")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--repo=")
    two_word_flags+=("--repo")
    local_nonpersistent_flags+=("--repo")
    local_nonpersistent_flags+=("--repo=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_module_help()
{
    last_command="rain_module_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_rain_module_install()
{
    last_command="rain_module_install"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--domain=")
    two_word_flags+=("--domain")
    local_nonpersistent_flags+=("--domain")
    local_nonpersistent_flags+=("--domain=")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--repo=")
    two_word_flags+=("--repo")
    local_nonpersistent_flags+=("--repo")
    local_nonpersistent_flags+=("--repo=")
    flags+=("--version=")
    two_word_flags+=("--version")
    local_nonpersistent_flags+=("--version")
    local_nonpersistent_flags+=("--version=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_module_publish()
{
    last_command="rain_module_publish"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--domain=")
    two_word_flags+=("--domain")
    local_nonpersistent_flags+=("--domain")
    local_nonpersistent_flags+=("--domain=")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--repo=")
    two_word_flags+=("--repo")
    local_nonpersistent_flags+=("--repo")
    local_nonpersistent_flags+=("--repo=")
    flags+=("--version=")
    two_word_flags+=("--version")
    local_nonpersistent_flags+=("--version")
    local_nonpersistent_flags+=("--version=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_module()
{
    last_command="rain_module"

    command_aliases=()

    commands=()
    commands+=("bootstrap")
    commands+=("help")
    commands+=("install")
    commands+=("publish")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_pkg()
{
    last_command="rain_pkg"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--datamodel")
    local_nonpersistent_flags+=("--datamodel")
    flags+=("--debug")
    local_nonpersistent_flags+=("--debug")
    flags+=("--experimental")
    flags+=("-x")
    local_nonpersistent_flags+=("--experimental")
    local_nonpersistent_flags+=("-x")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--node-style=")
    two_word_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style")
    local_nonpersistent_flags+=("--node-style=")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-owner=")
    two_word_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_rm()
{
    last_command="rain_rm"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--changeset")
    flags+=("-c")
    local_nonpersistent_flags+=("--changeset")
    local_nonpersistent_flags+=("-c")
    flags+=("--detach")
    flags+=("-d")
    local_nonpersistent_flags+=("--detach")
    local_nonpersistent_flags+=("-d")
    flags+=("--experimental")
    local_nonpersistent_flags+=("--experimental")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--role-arn=")
    two_word_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn")
    local_nonpersistent_flags+=("--role-arn=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_stackset_deploy()
{
    last_command="rain_stackset_deploy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--accounts=")
    two_word_flags+=("--accounts")
    local_nonpersistent_flags+=("--accounts")
    local_nonpersistent_flags+=("--accounts=")
    flags+=("--admin")
    local_nonpersistent_flags+=("--admin")
    flags+=("--config=")
    two_word_flags+=("--config")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    local_nonpersistent_flags+=("-c")
    flags+=("--detach")
    flags+=("-d")
    local_nonpersistent_flags+=("--detach")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--ignore-stack-instances")
    flags+=("-i")
    local_nonpersistent_flags+=("--ignore-stack-instances")
    local_nonpersistent_flags+=("-i")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--regions=")
    two_word_flags+=("--regions")
    local_nonpersistent_flags+=("--regions")
    local_nonpersistent_flags+=("--regions=")
    flags+=("--s3-bucket=")
    two_word_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket")
    local_nonpersistent_flags+=("--s3-bucket=")
    flags+=("--s3-owner=")
    two_word_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner")
    local_nonpersistent_flags+=("--s3-owner=")
    flags+=("--s3-prefix=")
    two_word_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix")
    local_nonpersistent_flags+=("--s3-prefix=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_stackset_help()
{
    last_command="rain_stackset_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_rain_stackset_ls()
{
    last_command="rain_stackset_ls"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--admin")
    local_nonpersistent_flags+=("--admin")
    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_stackset_rm()
{
    last_command="rain_stackset_rm"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--admin")
    local_nonpersistent_flags+=("--admin")
    flags+=("--detach")
    flags+=("-d")
    local_nonpersistent_flags+=("--detach")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_stackset()
{
    last_command="rain_stackset"

    command_aliases=()

    commands=()
    commands+=("deploy")
    commands+=("help")
    commands+=("ls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="ls"
    fi
    commands+=("rm")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("delete")
        aliashash["delete"]="rm"
        command_aliases+=("remove")
        aliashash["remove"]="rm"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-colour")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--debug")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_tree()
{
    last_command="rain_tree"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--both")
    flags+=("-b")
    local_nonpersistent_flags+=("--both")
    local_nonpersistent_flags+=("-b")
    flags+=("--dot")
    flags+=("-d")
    local_nonpersistent_flags+=("--dot")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_watch()
{
    last_command="rain_watch"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--region=")
    two_word_flags+=("--region")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    local_nonpersistent_flags+=("-r")
    flags+=("--wait")
    flags+=("-w")
    local_nonpersistent_flags+=("--wait")
    local_nonpersistent_flags+=("-w")
    flags+=("--debug")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_rain_root_command()
{
    last_command="rain"

    command_aliases=()

    commands=()
    commands+=("bootstrap")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("bootstrap")
        aliashash["bootstrap"]="bootstrap"
    fi
    commands+=("build")
    commands+=("cat")
    commands+=("cc")
    commands+=("console")
    commands+=("deploy")
    commands+=("diff")
    commands+=("fmt")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("format")
        aliashash["format"]="fmt"
    fi
    commands+=("forecast")
    commands+=("help")
    commands+=("info")
    commands+=("logs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("log")
        aliashash["log"]="logs"
    fi
    commands+=("ls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="ls"
    fi
    commands+=("merge")
    commands+=("module")
    commands+=("pkg")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("package")
        aliashash["package"]="pkg"
    fi
    commands+=("rm")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("del")
        aliashash["del"]="rm"
        command_aliases+=("delete")
        aliashash["delete"]="rm"
        command_aliases+=("remove")
        aliashash["remove"]="rm"
    fi
    commands+=("stackset")
    commands+=("tree")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("graph")
        aliashash["graph"]="tree"
    fi
    commands+=("watch")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--debug")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-colour")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

__start_rain()
{
    local cur prev words cword split
    declare -A flaghash 2>/dev/null || :
    declare -A aliashash 2>/dev/null || :
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -s || return
    else
        __rain_init_completion -n "=" || return
    fi

    local c=0
    local flag_parsing_disabled=
    local flags=()
    local two_word_flags=()
    local local_nonpersistent_flags=()
    local flags_with_completion=()
    local flags_completion=()
    local commands=("rain")
    local command_aliases=()
    local must_have_one_flag=()
    local must_have_one_noun=()
    local has_completion_function=""
    local last_command=""
    local nouns=()
    local noun_aliases=()

    __rain_handle_word
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_rain rain
else
    complete -o default -o nospace -F __start_rain rain
fi

# ex: ts=4 sw=4 et filetype=sh
