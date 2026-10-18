# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""The catalogues of shared/i18n, and how the application formats them."""

import os
from collections.abc import Sequence

import pytest

from support import SHARED
from vfox_plus.core.i18n.localizations import (
    Localizations,
    load_catalogue,
    message_keys,
    resolve_language,
    system_locales,
)
from vfox_plus.core.i18n.message_format import (
    Argument,
    MessageFormatError,
    MessagePart,
    Plural,
    format_message,
    parse_message,
)
from vfox_plus.core.settings.app_settings import LanguagePreference

I18N = SHARED / "i18n"
EN = load_catalogue(I18N, "en")
ZH = load_catalogue(I18N, "zh")


def _placeholders(parts: Sequence[MessagePart]) -> set[str]:
    names: set[str] = set()
    for part in parts:
        if isinstance(part, Argument):
            names.add(part.name)
        elif isinstance(part, Plural):
            names.add(part.name)
            for nested in part.cases.values():
                names |= _placeholders(nested)
    return names


def test_formats_placeholders_and_plurals_with_exact_matches_first() -> None:
    parts = parse_message("{count, plural, =0{none} =1{one {name}} other{{count} of {name}}}!")
    assert format_message(parts, {"count": 0, "name": "x"}, "en") == "none!"
    assert format_message(parts, {"count": 1, "name": "x"}, "en") == "one x!"
    assert format_message(parts, {"count": 7, "name": "x"}, "en") == "7 of x!"
    assert format_message(parts, {"count": 2.5, "name": "x"}, "en") == "2.5 of x!"
    assert format_message(parts, {"count": "oops", "name": "x"}, "en") == "none!"


def test_an_empty_exact_case_is_still_chosen() -> None:
    assert format_message(parse_message("{n, plural, =0{} other{some}}"), {"n": 0}, "en") == ""


def test_chooses_plural_categories_by_language() -> None:
    parts = parse_message("{n, plural, one{single} other{many}}")
    assert format_message(parts, {"n": 1}, "en") == "single"
    assert format_message(parts, {"n": 1.0}, "en") == "single"
    assert format_message(parts, {"n": 1}, "zh") == "many"


def test_keeps_unknown_placeholders_visible_and_apostrophes_literal() -> None:
    assert format_message(parse_message("don't {x}"), {}, "en") == "don't {x}"
    assert format_message(parse_message("{x}"), {"x": 2.0}, "en") == "2"


@pytest.mark.parametrize("source", ["{a, select, x{y} other{z}}", "{a, plural, =1{y}}", "a } b", "{1}", "{a, plural x"])
def test_rejects_malformed_messages(source: str) -> None:
    with pytest.raises(MessageFormatError):
        parse_message(source)


def test_the_catalogues_are_the_only_ones_and_the_template_is_english() -> None:
    assert sorted(os.listdir(I18N)) == ["app_en.arb", "app_zh.arb"]
    assert EN["@@locale"] == "en"
    assert ZH["@@locale"] == "zh"


def test_the_catalogues_have_the_same_messages() -> None:
    assert sorted(message_keys(ZH)) == sorted(message_keys(EN))


def test_every_message_of_the_template_is_described() -> None:
    for key in message_keys(EN):
        assert EN.get(f"@{key}", {}).get("description"), key


def test_messages_parse_and_translations_keep_every_placeholder() -> None:
    for key in message_keys(EN):
        template = _placeholders(parse_message(EN[key]))
        translated = _placeholders(parse_message(ZH[key]))
        assert template <= translated, f"{key} lacks placeholders in app_zh.arb"
        declared = set(EN.get(f"@{key}", {}).get("placeholders", {}))
        assert template == declared, key


def test_localizations_format_messages_of_each_language_and_fall_back_to_english() -> None:
    english = Localizations("en", EN, EN)
    chinese = Localizations("zh", ZH, EN)
    assert english.t("sdksVersionCount", count=2) == "2 versions installed"
    assert english.t("sdksVersionCount", count=1) == "1 version installed"
    assert chinese.t("sdksVersionCount", count=2) == "已安装 2 个版本"
    assert chinese.t("taskInstall", subject="maven@3.9.16") == "安装 maven@3.9.16"
    assert Localizations("zh", {"navHome": "首页"}, EN).t("navSdks") == "SDKs"
    assert english.t("noSuchKey") == "noSuchKey"
    assert english.format("sdksVersionCount", {"count": 3}) == "3 versions installed"


def test_resolves_the_system_language() -> None:
    assert resolve_language(LanguagePreference.SYSTEM, ["zh-TW"]) == "zh"
    assert resolve_language(LanguagePreference.SYSTEM, ["zh_CN.UTF-8"]) == "zh"
    assert resolve_language(LanguagePreference.SYSTEM, ["de-DE", "en-US"]) == "en"
    assert resolve_language(LanguagePreference.SYSTEM, ["fr"]) == "en"
    assert resolve_language(LanguagePreference.ENGLISH, ["zh-CN"]) == "en"
    assert resolve_language(LanguagePreference.SIMPLIFIED_CHINESE, ["en"]) == "zh"


def test_reads_the_desktop_locales_in_the_order_of_glib() -> None:
    environment = {"LANGUAGE": "zh_CN:en", "LC_ALL": "", "LC_MESSAGES": "C.UTF-8", "LANG": "de_DE.UTF-8"}
    assert system_locales(environment) == ["zh_CN", "en", "de_DE.UTF-8"]
    assert system_locales({"LANG": "C"}) == []
    assert system_locales({"LANG": "POSIX"}) == []
