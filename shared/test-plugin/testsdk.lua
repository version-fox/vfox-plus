-- Copyright 2026 Han Li and contributors
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--   http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- An offline vfox plugin for end-to-end tests of vfox-plus.
--
-- It offers two versions of a tiny SDK and downloads them from the local test
-- server whose base URL is in the VFOX_PLUS_TEST_SERVER environment variable.
-- The server protocol is described in README.md next to this file.

PLUGIN = {
    name = "testsdk",
    version = "0.1.0",
    homepage = "https://github.com/version-fox/vfox-plus",
    license = "Apache-2.0",
    description = "An offline SDK for the vfox-plus end-to-end tests.",
    minRuntimeVersion = "1.0.0",
    notes = {},
}

local function server()
    local url = os.getenv("VFOX_PLUS_TEST_SERVER")
    if url == nil or url == "" then
        error("VFOX_PLUS_TEST_SERVER is not set")
    end
    return url
end

function PLUGIN:Available(ctx)
    return {
        {
            version = "2.0.0",
            note = "LTS",
            addition = { { name = "testtool", version = "2.0" } },
        },
        { version = "1.0.0" },
    }
end

function PLUGIN:PreInstall(ctx)
    return {
        version = ctx.version,
        url = server() .. "/sdk/testsdk-" .. ctx.version .. ".tar.gz",
    }
end

function PLUGIN:EnvKeys(ctx)
    return { { key = "PATH", value = ctx.path .. "/bin" } }
end
