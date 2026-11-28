// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !e2e

package main

import (
	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/core/profile/builtin"
)

// onlineProfiles is the online directory of profiles, cached in the cache
// directory of the user; nil without one.
func onlineProfiles() *profile.Catalog {
	directory, err := core.UserCacheSubdir("vfox-plus", "profiles")
	if err != nil {
		return nil
	}
	return &profile.Catalog{Directory: directory, Shipped: builtin.Files()}
}
