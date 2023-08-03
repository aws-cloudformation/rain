#!/usr/bin/env bash

if [ "${BASH_VERSINFO:-0}" -lt "4" ]; then
    >&2 echo "Your bash version does not support associative arrays. Please update to use this script"
    exit 1
fi

DOC_BASE="https://docs.aws.amazon.com/serverless-application-model/latest/developerguide"

declare -A types=(
    [String]=Primitive
    [Integer]=Primitive
    [Double]=Primitive
    [Long]=Primitive
    [Boolean]=Primitive
    [Timestamp]=Primitive
    [Json]=Primitive
    [Map]=Composite
    [List]=Composite
)

d=$(mktemp -d)
git clone --branch main https://github.com/awsdocs/aws-sam-developer-guide.git $d

cd "$d"

#  The Name property was declared twice.
# echo "diff --git a/doc_source/sam-property-function-eventbridgerule.md b/doc_source/sam-property-function-eventbridgerule.md
# index 7a88e81..1d4a8ac 100644
# --- a/doc_source/sam-property-function-eventbridgerule.md
# +++ b/doc_source/sam-property-function-eventbridgerule.md
# @@ -14,7 +14,6 @@ To declare this entity in your AWS Serverless Application Model \(AWS SAM\) temp
#    [DeadLetterConfig](#sam-function-eventbridgerule-deadletterconfig): DeadLetterConfig
#    [EventBusName](#sam-function-eventbridgerule-eventbusname): String
#    [Input](#sam-function-eventbridgerule-input): String
# -  [Name](#sam-function-eventbridgerule-name): String
#    [InputPath](#sam-function-eventbridgerule-inputpath): String
#    [Name](#sam-function-eventbridgerule-name): String
#    [Pattern](#sam-function-eventbridgerule-pattern): [EventPattern](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-eventpattern)
# " | git apply

echo "ResourceSpecificationVersion: $(git rev-parse HEAD)"

cd doc_source

declare -A prefix_types
declare -A completed

# Resource types
echo "ResourceTypes:"

for file in sam-resource-*.md; do
    first="$(grep -n '```' $file | head -n1 | cut -d: -f1)"
    second="$(grep -n '```' $file | head -n2 | tail -n1 | cut -d: -f1)"

    resource_name="$(head -n$((first+1)) $file | tail -n1 | cut -d" " -f2)"
    if [ "$resource_name" == "<source-resource-logical-id>:" ]; then
        continue
    fi

    echo "  $resource_name:"

    file_base=$(basename -s.md $file)
    echo "    Documentation: ${DOC_BASE}/${file_base}.html"

    echo "    Properties:"

    IFS=$'\n'
    for line in $(head -n$((second-1)) $file | tail -n$((second-first-3))); do
        prop_name="$(echo "$line" | grep -o '[A-Z]\w\+' | head -n1)"
        if [ "$prop_name" == "" ]; then
            >&2 echo "Line in $resource_name has no property: $line"
            continue
        fi
        echo "      $prop_name:"

        echo "        Documentation: ${DOC_BASE}/${file_base}.html#${file_base/resource-/}-${prop_name,,}"

        prop_type=$(echo "$line" | cut -d: -f2 | awk -F"|" '{print $NF}' | sed -e 's/^ *\[//' -e 's/\].*$//' | xargs)

        if [ "$prop_type" != "" ]; then
            if [ "${types[$prop_type]}" == "Primitive" ]; then
                echo "        PrimitiveType: $prop_type"
            elif [ "${types[$prop_type]}" == "Composite" ]; then
                echo "        Type: $prop_type"
                echo "        PrimitiveItemType: String"
            else
                echo "        Type: $prop_type"
            fi
        else 
            >&2 echo "prop_type in $resource_name not set for line: $line"
            continue
        fi

        # Find out if it's required
        mention=$(grep -n "\`$prop_name\`" $file | head -n1 | cut -d: -f1 | xargs)
        required=$(tail -n+${mention} "$file" | grep "*Required*" | head -n1 | cut -d: -f2 | xargs)
        if [ "$required" == "Yes" ]; then
            echo "        Required: True"
        else
            echo "        Required: False"
        fi
    done

    echo "    Attributes:"



    # Store the type name with the prefix
    prefix_types[$(basename -s .md $file | cut -d- -f3)]=$resource_name
done

# Property types
echo "PropertyTypes:"

for file in sam-property-*.md; do
    resource_name=${prefix_types[$(basename -s .md $file | cut -d- -f3)]}

    prop_type_name="$(head -n1 $file | sed -e 's/^# //' -e 's/<.*$//')"

    if [ -n "${completed[${resource_name}::${prop_type_name}]}" ]; then
        continue
    fi

    echo "  ${resource_name}.${prop_type_name}:"

    file_base=$(basename -s.md $file)
    echo "    Documentation: ${DOC_BASE}/${file_base}.html"

    echo "    Properties:"

    first="$(grep -n '```' $file | head -n1 | cut -d: -f1)"
    second="$(grep -n '```' $file | head -n2 | tail -n1 | cut -d: -f1)"

    IFS=$'\n'
    for line in $(head -n$((second-1)) $file | tail -n$((second-first-1))); do
        prop_name="$(echo "$line" | grep -o '[A-Z]\w\+' | head -n1)"
        echo "      $prop_name:"

        echo "        Documentation: ${DOC_BASE}/${file_base}.html#${file_base/property-/}-${prop_name,,}"

        prop_type=$(echo "$line" | cut -d: -f2 | awk -F"|" '{print $NF}' | sed -e 's/^ *\[//' -e 's/\].*$//' | xargs)

        if [ "$prop_type" != "" ]; then
            if [ "${types[$prop_type]}" == "Primitive" ]; then
                echo "        PrimitiveType: $prop_type"
            elif [ "${types[$prop_type]}" == "Composite" ]; then
                echo "        Type: $prop_type"
                echo "        PrimitiveItemType: String"
            else
                echo "        Type: $prop_type"
            fi
        else
            >&2 echo "prop_type not set for line: $line"
        fi

        # Find out if it's required
        mention=$(grep -n "\`$prop_name\`" $file | head -n1 | cut -d: -f1 | xargs)
        required=$(tail -n+${mention} "$file" | grep "*Required*" | head -n1 | cut -d: -f2 | xargs)
        if [ $required == "Yes" ]; then
            echo "        Required: True"
        else
            echo "        Required: False"
        fi
    done

    completed[${resource_name}::${prop_type_name}]="yes"
done

rm -rf "$d"
