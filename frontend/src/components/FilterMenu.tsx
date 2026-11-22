import { Check, ChevronDown, ListFilter } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { Fragment } from 'react'
import menu from './Menu.module.css'
import styles from './FilterMenu.module.css'

interface FilterSection {
  label: string
  value: string
  options: readonly { value: string; label: string }[]
  onChange: (value: string) => void
}

interface FilterMenuProps {
  label: string
  active: boolean
  sections: FilterSection[]
}

export function FilterMenu({ label, active, sections }: FilterMenuProps) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className={styles.trigger} data-active={active || undefined}>
          <ListFilter size={13} />
          <span>{label}</span>
          <ChevronDown size={12} className={styles.chevron} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={menu.content} align="end" sideOffset={4}>
          {sections.map((section, i) => (
            <Fragment key={section.label}>
              {i > 0 && <DropdownMenu.Separator className={menu.separator} />}
              <DropdownMenu.Label className={menu.label}>{section.label}</DropdownMenu.Label>
              <DropdownMenu.RadioGroup value={section.value} onValueChange={section.onChange}>
                {section.options.map((option) => (
                  <DropdownMenu.RadioItem
                    key={option.value}
                    value={option.value}
                    className={menu.item}
                    onSelect={(e) => e.preventDefault()}
                  >
                    <span className={menu.check}>
                      <DropdownMenu.ItemIndicator>
                        <Check size={13} strokeWidth={3} />
                      </DropdownMenu.ItemIndicator>
                    </span>
                    {option.label}
                  </DropdownMenu.RadioItem>
                ))}
              </DropdownMenu.RadioGroup>
            </Fragment>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
