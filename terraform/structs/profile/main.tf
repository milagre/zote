terraform {
  experiments = [module_variable_optional_attrs]
}

variable "profile" {
  type = object({
    cpu = object({
      min = string
      max = string
    })
    mem = object({
      min = string
      max = string
    })
    num = optional(object({
      min = number
      max = number
    }))
  })

  // CPU

  validation {
    condition     = can(regex("^[0-9]+m$", var.profile.cpu.min))
    error_message = "cpu minimum must be a number ending in 'm'"
  }

  validation {
    condition     = can(regex("^[0-9]+m$", var.profile.cpu.max))
    error_message = "cpu maximum must be a number ending in 'm'"
  }

  validation {
    condition     = can(tonumber(trim(var.profile.cpu.max, "m")) < tonumber(trim(var.profile.cpu.min, "m")))
    error_message = "cpu maximum must be greater or equal to cpu minimum"
  }

  // Memory

  validation {
    condition     = can(regex("^[0-9]+M$", var.profile.mem.min))
    error_message = "memory minimum must be a number ending in 'M'"
  }

  validation {
    condition     = can(regex("^[0-9]+M$", var.profile.mem.max))
    error_message = "memory maximum must be a number ending in 'M'"
  }

  validation {
    condition     = can(tonumber(trim(var.profile.mem.max, "M")) < tonumber(trim(var.profile.mem.min, "M")))
    error_message = "mem maximum must be greater or equal to mem minimum"
  }

  // Num

  validation {
    condition     = can(var.profile.num.max < var.profile.num.min)
    error_message = "num maximum must be greater or equal to num minimum"
  }
}

output "cpu_cores" {
  value = {
    min = tonumber(trim(var.profile.cpu.min, "m")) / 1000.0
    max = tonumber(trim(var.profile.cpu.max, "m")) / 1000.0
  }
}

output "mem_mb" {
  value = {
    min = tonumber(trim(var.profile.mem.min, "M"))
    max = tonumber(trim(var.profile.mem.max, "M"))
  }
}

output "num" {
  value = {
    min = tonumber(var.profile.num.min)
    max = tonumber(var.profile.num.max)
  }
}
