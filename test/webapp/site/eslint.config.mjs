import globals from "globals";
import pluginJs from "@eslint/js";


export default [
    {languageOptions: { globals: globals.browser }},
    pluginJs.configs.recommended,
    {ignores: ["dist", "config-sample.js", "js/vendor"]}
];
