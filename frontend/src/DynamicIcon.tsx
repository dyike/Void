import { icons } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface DynamicIconProps {
    name?: string
    size?: number
    className?: string
    fallback?: string
}

export const DynamicIcon = ({ name, size = 15, className, fallback = 'Box' }: DynamicIconProps) => {
    const FallbackIcon = (icons as any)[fallback] as LucideIcon || icons.Box
    if (!name) return <FallbackIcon size={size} className={className} />

    const IconComponent = (icons as any)[name] as LucideIcon
    if (!IconComponent) return <FallbackIcon size={size} className={className} />

    return <IconComponent size={size} className={className} />
}
