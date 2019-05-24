// Copyright (C) 2018 go-dacc authors
//
// This file is part of the go-dacc library.
//
// the go-dacc library is free software: you can redistribute it and/or
// modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dacc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dacc library.  If not, see
// <http://www.gnu.org/licenses/>.
//
#include "cmd/dummy_neb/cli/pkg.h"
#include "cmd/dummy_neb/generator/generator_base.h"
#include "util/wakeable_queue.h"

class cli_generator : public generator_base {
public:
  cli_generator();
  virtual ~cli_generator();

  void update_info(generate_block *block);
  inline void append_pkg(const std::shared_ptr<ff::net::package> &pkg) {
    m_pkgs.push_back(pkg);
  }

  virtual std::shared_ptr<corepb::Account> gen_account();
  virtual std::shared_ptr<corepb::Transaction> gen_tx();
  virtual checker_tasks::task_container_ptr_t gen_tasks();

  address_t m_auth_admin_addr;
  address_t m_nr_admin_addr;
  address_t m_dip_admin_addr;

protected:
  neb::util::wakeable_queue<std::shared_ptr<ff::net::package>> m_pkgs;
};