import { Switch as RadixSwitch } from 'radix-ui'
import styles from './Switch.module.css'

type SwitchProps = RadixSwitch.SwitchProps

export function Switch(props: SwitchProps) {
  return (
    <RadixSwitch.Root className={styles.root} {...props}>
      <RadixSwitch.Thumb className={styles.thumb} />
    </RadixSwitch.Root>
  )
}
